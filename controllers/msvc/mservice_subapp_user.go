package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	pb "github.com/gaterace/mservice/pkg/mserviceaccount"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

// *********************************
// local funcs to get mservice data
// *********************************

func getAccountUsers(client pb.MServiceAccountClient, mctx context.Context, acctname string) *pb.GetAccountUsersResponse {
	fmt.Println("in getAccountUsers func")
	req := pb.GetAccountUsersRequest{}
	req.AccountName = acctname
	resp, err := client.GetAccountUsers(mctx, &req)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	return resp
}

func getAccountRoles(client pb.MServiceAccountClient, mctx context.Context, acctId int64) *pb.GetAccountRolesResponse {
	fmt.Println("in getAccountRoles func")
	req := pb.GetAccountRolesRequest{}
	req.AccountId = acctId
	resp, err := client.GetAccountRoles(mctx, &req)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	return resp
}

func getAvailableRoles(user pb.AccountUser, acctRoles *pb.GetAccountRolesResponse) []*pb.AccountRole {
	fmt.Println("in getAvailableRoles func")

	var available_roles []*pb.AccountRole
	var assigned_claim_ids []int64

	// get unique assigned claim name ids
	for _, assigned := range user.AccountRoles {
		// fmt.Println("assigned = ", assigned.ClaimNameId, assigned.Claim.ClaimName, assigned.ClaimVal)
		for _, role_claims := range assigned.ClaimValues {
			assigned_claim_ids = append(assigned_claim_ids, role_claims.ClaimNameId)
		}
		// assigned_claim_ids = append(assigned_claim_ids, assigned.ClaimValues)
	}
	// already unique?
	unique_assigned_claim_ids := lo.Uniq(assigned_claim_ids)
	// unique_assigned_ids := assigned_role_ids
	fmt.Println("unique assigned ids", unique_assigned_claim_ids)

	// now filter the available claim name id's
	for _, avail := range acctRoles.AccountRoles {
		include := true
		// fmt.Println("available = ", avail.ClaimNameId, avail.ClaimName)

		for _, unique_assigned_id := range unique_assigned_claim_ids {
			for _, included_claims := range avail.ClaimValues {
				if unique_assigned_id == included_claims.ClaimNameId {
					// fmt.Println("match! on", unique_assigned_id, avail.ClaimNameId)
					include = false
				}
			}
		}
		if include {
			available_roles = append(available_roles, avail)
		}
	}

	return available_roles

}

//*********************************************************
//
//	MService -> My User Account / Change My Password
//
//*********************************************************

// Load me page
func GetMyUserRecord(c *gin.Context) {
	// fmt.Println("in getAccounts1")
	session := sessions.Default(c)
	session.Set("working_actname", session.Get("actname"))
	session.Save()
	// fmt.Println("your account is :", session.Get("actname"))

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	resp := getAccountUserById(client, session.Get("uid").(int64), mctx)

	c.HTML(http.StatusOK,
		"me_page.html",
		gin.H{
			"title":    "MService",
			"active":   "me",
			"subtitle": "Change Your User Record",
			"auth":     c.GetBool("auth"),
			"claims":   c.GetStringMap("claims"),
			"fullname": session.Get("fullname"),
			"user":     resp,
		},
	)
	// }

}

func GetChangeMyPasswordForm(c *gin.Context) {
	// fmt.Println("in getAccounts1")
	session := sessions.Default(c)
	session.Set("working_actname", session.Get("actname"))
	session.Save()
	// fmt.Println("your account is :", session.Get("actname"))

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	resp := getAccountUserById(client, session.Get("uid").(int64), mctx)

	c.HTML(http.StatusOK,
		"me_page.html",
		gin.H{
			"title":    "MService",
			"active":   "password",
			"subtitle": "Change Your Password",
			"auth":     c.GetBool("auth"),
			"claims":   c.GetStringMap("claims"),
			"fullname": session.Get("fullname"),
			"user":     resp,
		},
	)
	// }

}

//*********************************************************
//
//	MService -> Users subapp
//
//*********************************************************

// Get User List - hx-target="#sub_app_content"
func GetSubappUsers(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("b4 working acct?", session.Get("working_actname"))
	fmt.Println("b4 acct?", session.Get("actname"))

	if session.Get("working_actname") == nil {
		session.Set("working_actname", session.Get("actname"))
		session.Save()
	}
	fmt.Println("working account is :", session.Get("working_actname"))

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	var acctnames []string
	var workingAccountId int64
	resp1 := getAccountByName(client, mctx, session.Get("working_actname").(string))
	workingAccountId = resp1.GetAccount().AccountId

	if session.Get("actname") == "master" {
		resp := getAccountNames(client, mctx)
		acctnames = resp.GetAccountName()
	}

	resp := getAccountUsers(client, mctx, session.Get("working_actname").(string))

	c.HTML(http.StatusOK,
		"subapp_partial.html",
		gin.H{
			"auth":               c.GetBool("auth"),
			"claims":             c.GetStringMap("claims"),
			"sub_title":          "Mservice Account Users",
			"sub_active":         "users",
			"users":              resp,
			"acctnames":          acctnames,
			"acctname":           session.Get("working_actname").(string),
			"working_account_id": workingAccountId,
		},
	)

}

// Get different Account User List - hx-target="#action_content"
func GetUsersByAcctname(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("b4 working acct?", session.Get("working_actname"))
	fmt.Println("b4 acct?", session.Get("actname"))

	acctname := c.Request.URL.Query()["acctname"][0]
	fmt.Println("****working account*****", acctname)
	session.Set("working_actname", acctname)
	session.Save()

	fmt.Println("working account is :", session.Get("working_actname"))

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	resp := getAccountUsers(client, mctx, session.Get("working_actname").(string))

	// get working acct id
	var workingAccountId int64
	resp1 := getAccountByName(client, mctx, session.Get("working_actname").(string))
	workingAccountId = resp1.GetAccount().AccountId

	c.HTML(http.StatusOK,
		"user_list.html",
		gin.H{
			"auth":               c.GetBool("auth"),
			"claims":             c.GetStringMap("claims"),
			"users":              resp,
			"acctname":           session.Get("working_actname").(string),
			"working_account_id": workingAccountId,
		},
	)

}

// Get User Form - hx-target="#action_content"
func GetAccountUserById(c *gin.Context) {
	// session := sessions.Default(c)
	var working_account_id int64

	fmt.Println("raw user id:", c.Param("id"))

	if c.Param("id") != "new" {
		userId, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		conn := getGrpcConn()
		defer conn.Close()

		client := pb.NewMServiceAccountClient(conn)

		mctx := getMctxFromCookie(c)

		resp := getAccountUserById(client, userId, mctx)

		c.HTML(http.StatusOK,
			"user_form.html",
			gin.H{
				"auth":   c.GetBool("auth"),
				"claims": c.GetStringMap("claims"),
				"user":   resp,
			},
		)

	} else { // new user rec
		fmt.Println("raw acctid param............", c.Request.URL.Query()["acctid"][0])

		i, err := strconv.ParseInt(c.Request.URL.Query()["acctid"][0], 10, 64)
		if err != nil {
			panic(err)
		} else {
			working_account_id = i
		}
		fmt.Println("working acctid for new:::", working_account_id)

		c.HTML(http.StatusOK,
			"user_form.html",
			gin.H{
				"auth":               c.GetBool("auth"),
				"claims":             c.GetStringMap("claims"),
				"isCreate":           true,
				"working_account_id": working_account_id,
			},
		)

	}

}

// Update User and redisplay user form - hx-target="#action_content"
func UpdateUser(c *gin.Context) {
	me := false

	if len(c.Request.URL.Query()["me"]) > 0 {
		if c.Request.URL.Query()["me"][0] == "true" {
			me = true
		}
	}

	session := sessions.Default(c)
	fmt.Println("update current working acct:", session.Get("working_actname"))

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.UpdateAccountUserRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := client.UpdateAccountUser(mctx, &req)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update account error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"user_form.html",
			gin.H{
				"user":    req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
			},
		)

	} else {
		// reget the user by id
		resp1 := getAccountUserById(client, req.UserId, mctx)

		if !me {
			c.HTML(http.StatusOK,
				"user_form.html",
				gin.H{
					"auth":    c.GetBool("auth"),
					"claims":  c.GetStringMap("claims"),
					"user":    resp1,
					"success": true,
				},
			)

		} else {
			c.HTML(http.StatusOK,
				"me_user_form.html",
				gin.H{
					"auth":    c.GetBool("auth"),
					"claims":  c.GetStringMap("claims"),
					"user":    resp1,
					"success": true,
				},
			)

		}
	}

}

// Create new User and display user form - hx-target="#action_content"
func CreateUser(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("update current working acct:", session.Get("working_actname"))

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.CreateAccountUserRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := client.CreateAccountUser(mctx, &req)
	// fmt.Println(resp)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update account error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"user_form.html",
			gin.H{
				"user":    req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
			},
		)

	} else {
		// reget the user by id
		resp1 := getAccountUserById(client, resp.UserId, mctx)

		c.HTML(http.StatusOK,
			"user_form.html",
			gin.H{
				"auth":    c.GetBool("auth"),
				"claims":  c.GetStringMap("claims"),
				"user":    resp1,
				"success": true,
			},
		)
	}

}

func DeleteUser(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("delete user current working acct:", session.Get("working_actname"))

	userId, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	// var version int32
	version, _ := strconv.ParseInt(c.Param("version"), 10, 32)

	acctname := session.Get("actname").(string)

	fmt.Println("id:", userId)
	// fmt.Println("version:", version)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// Delete the record
	req := pb.DeleteAccountUserRequest{}
	req.UserId = userId
	req.Version = int32(version)
	resp, err := client.DeleteAccountUser(mctx, &req)
	if err == nil {
		jtext, err := json.MarshalIndent(resp, "", "  ")
		if err == nil {
			fmt.Println(string(jtext))
		}
	}

	// // if ok set working_actname to actname get account names
	var acctnames []string
	if acctname == "master" {
		resp := getAccountNames(client, mctx)
		acctnames = resp.GetAccountName()
	}

	// then get working actname users
	resp1 := getAccountUsers(client, mctx, session.Get("working_actname").(string))

	fmt.Println("delete before render", session.Get("working_actname"))

	// get working acct id
	var workingAccountId int64

	resp2 := getAccountByName(client, mctx, session.Get("working_actname").(string))
	workingAccountId = resp2.GetAccount().AccountId

	c.HTML(http.StatusOK,
		"users.html",
		gin.H{
			"auth":               c.GetBool("auth"),
			"claims":             c.GetStringMap("claims"),
			"users":              resp1,
			"working_account_id": workingAccountId,
			"acctnames":          acctnames,
		},
	)

}

// Get change/reset Password Form - hx-target="#action_content"
func GetPasswordReset(c *gin.Context) {
	// get the user record to get the version and name to display on the form
	fmt.Println("raw user id:", c.Param("id"))
	userId, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	resp := getAccountUserById(client, userId, mctx)

	//display the form
	c.HTML(http.StatusOK,
		"password_form.html",
		gin.H{
			"auth":   c.GetBool("auth"),
			"claims": c.GetStringMap("claims"),
			"user":   resp,
			"reset":  true,
		},
	)

}

func ResetPassword(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("reset password current working acct:", session.Get("working_actname"))

	// acctname := session.Get("actname").(string)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// reset password
	req := pb.ResetAccountUserPasswordRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := client.ResetAccountUserPassword(mctx, &req)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update account error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"password_form.html",
			gin.H{
				"user":    req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
				"reset":   true,
			},
		)

	} else {
		resp1 := getAccountUserById(client, req.UserId, mctx)

		c.HTML(http.StatusOK,
			"password_form.html",
			gin.H{
				"auth":    c.GetBool("auth"),
				"claims":  c.GetStringMap("claims"),
				"user":    resp1,
				"success": true,
				"reset":   true,
			},
		)
	}

}

func ChangeMyPassword(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("reset password current working acct:", session.Get("working_actname"))

	// acctname := session.Get("actname").(string)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// reset password
	req := pb.UpdateAccountUserPasswordRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// fmt.Println("change pw data", req)

	resp, err := client.UpdateAccountUserPassword(mctx, &req)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update password error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"me_password_form.html",
			gin.H{
				"user":    req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
				"reset":   true,
			},
		)

	} else {
		// reget the user by id
		resp1 := getAccountUserById(client, req.UserId, mctx)

		c.HTML(http.StatusOK,
			"me_password_form.html",
			gin.H{
				"auth":    c.GetBool("auth"),
				"claims":  c.GetStringMap("claims"),
				"user":    resp1,
				"success": true,
				"active":  "password",
			},
		)
	}

}

// Get Role claim assignment Form - hx-target="#action_content"
func GetAssignRolesToUserForm(c *gin.Context) {
	// session := sessions.Default(c)

	fmt.Println("raw user id:", c.Param("id"))

	userId, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// first get the user record w/assigned roles
	resp := getAccountUserById(client, userId, mctx)

	// next get all the available roles
	resp1 := getAccountRoles(client, mctx, resp.AccountUser.AccountId)

	// now compare available vs assigned
	available_roles := getAvailableRoles(*resp.AccountUser, resp1)

	c.HTML(http.StatusOK,
		"user_role_assignment_form.html",
		gin.H{
			"auth":      c.GetBool("auth"),
			"claims":    c.GetStringMap("claims"),
			"user":      resp,
			"available": available_roles,
		},
	)

}

// add a user to role then reGet user Role  assignment Form - hx-target="#action_content"
func AddUserToRole(c *gin.Context) {
	// session := sessions.Default(c)
	fmt.Println("********************* add user to role?")

	userId, _ := strconv.ParseInt(c.Param("user_id"), 10, 64)
	// var version int32
	roleId, _ := strconv.ParseInt(c.Param("role_id"), 10, 64)

	fmt.Println("userid:", userId)
	fmt.Println("roleId:", roleId)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// add the claim
	req0 := pb.AddUserToRoleRequest{}
	req0.RoleId = roleId
	req0.UserId = userId
	resp0, err0 := client.AddUserToRole(mctx, &req0)
	if err0 == nil {
		jtext, err := json.MarshalIndent(resp0, "", "  ")
		if err == nil {
			fmt.Println("resp from add user to role ********")
			fmt.Println(string(jtext))
		}
	}

	// now reget the user and available roles and redisplay the form

	// first get the user record w/assigned roles
	resp := getAccountUserById(client, userId, mctx)

	// next get all the available roles
	resp1 := getAccountRoles(client, mctx, resp.AccountUser.AccountId)

	// now compare available vs assigned
	available_roles := getAvailableRoles(*resp.AccountUser, resp1)

	c.HTML(http.StatusOK,
		"user_role_assignment_form.html",
		gin.H{
			"auth":      c.GetBool("auth"),
			"claims":    c.GetStringMap("claims"),
			"user":      resp,
			"available": available_roles,
		},
	)

}

// remove a user from role then reGet user Role  assignment Form - hx-target="#action_content"
func RemoveUserFromRole(c *gin.Context) {
	// session := sessions.Default(c)
	fmt.Println("********************* remove user from role?")

	userId, _ := strconv.ParseInt(c.Param("user_id"), 10, 64)
	// var version int32
	roleId, _ := strconv.ParseInt(c.Param("role_id"), 10, 64)

	// cctname := session.Get("actname").(string)

	fmt.Println("userid:", userId)
	fmt.Println("roleId:", roleId)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// add the claim
	req0 := pb.RemoveUserFromRoleRequest{}
	req0.RoleId = roleId
	req0.UserId = userId
	resp0, err0 := client.RemoveUserFromRole(mctx, &req0)
	if err0 == nil {
		jtext, err := json.MarshalIndent(resp0, "", "  ")
		if err == nil {
			fmt.Println("resp from remove user from role ********")
			fmt.Println(string(jtext))
		}
	}

	// now reget the user and available roles and redisplay the form

	// first get the user record w/assigned roles
	resp := getAccountUserById(client, userId, mctx)

	// next get all the available roles
	resp1 := getAccountRoles(client, mctx, resp.AccountUser.AccountId)

	// now compare available vs assigned
	available_roles := getAvailableRoles(*resp.AccountUser, resp1)

	c.HTML(http.StatusOK,
		"user_role_assignment_form.html",
		gin.H{
			"auth":      c.GetBool("auth"),
			"claims":    c.GetStringMap("claims"),
			"user":      resp,
			"available": available_roles,
		},
	)

}
