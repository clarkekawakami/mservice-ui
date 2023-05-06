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
	// "github.com/samber/lo"
)

// *********************************
// local funcs to get mservice data
// *********************************

func getAccountRoleById(client pb.MServiceAccountClient, mctx context.Context, id int64) *pb.GetAccountRoleByIdResponse {
	fmt.Println("in getAccountRoleById func")
	req := pb.GetAccountRoleByIdRequest{}
	req.RoleId = id
	resp, err := client.GetAccountRoleById(mctx, &req)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	return resp
}

func getAvailableClaims(role pb.AccountRole, claims *pb.GetClaimNamesResponse) []int64 {
	fmt.Println("in getAvailableClaims func")

	var available_claims []int64
	var assigned_claim_ids []int64

	// get unique assigned claim name ids
	for _, assigned := range role.ClaimValues {
		// fmt.Println("assigned = ", assigned.ClaimNameId, assigned.Claim.ClaimName, assigned.ClaimVal)
		assigned_claim_ids = append(assigned_claim_ids, assigned.ClaimNameId)
	}
	unique_assigned_ids := lo.Uniq(assigned_claim_ids)
	// fmt.Println("unique assigned ids", unique_assigned_ids)

	// now filter the available claim name id's
	for _, avail := range claims.Claims {
		include := true
		// fmt.Println("available = ", avail.ClaimNameId, avail.ClaimName)

		for _, unique_assigned_id := range unique_assigned_ids {
			if unique_assigned_id == avail.ClaimNameId {
				// fmt.Println("match! on", unique_assigned_id, avail.ClaimNameId)
				include = false
			}
		}
		if include {
			available_claims = append(available_claims, avail.ClaimNameId)
		}
	}

	return available_claims

}

//*********************************************************
//
//	MService -> Roles subapp
//
//*********************************************************

// Get Roles List - hx-target="#sub_app_content"
func GetSubappRoles(c *gin.Context) {
	session := sessions.Default(c)
	// fmt.Println("b4 working acct?", session.Get("working_actname"))
	// fmt.Println("b4 acct?", session.Get("actname"))

	// acctname := c.Request.URL.Query()["acctname"][0]
	// fmt.Println("****working account*****", acctname)
	// session.Set("working_actname", acctname)

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

	fmt.Println("acctnames", acctnames)

	fmt.Println("workingAccountId", workingAccountId)

	// get account roles
	resp := getAccountRoles(client, mctx, workingAccountId)

	c.HTML(http.StatusOK,
		"subapp_partial.html",
		gin.H{
			"auth":               c.GetBool("auth"),
			"claims":             c.GetStringMap("claims"),
			"sub_title":          "Mservice Account Roles",
			"sub_active":         "roles",
			"roles":              resp,
			"acctnames":          acctnames,
			"acctname":           session.Get("working_actname").(string),
			"working_account_id": workingAccountId,
		},
	)

}

// Get different Account Roles List - hx-target="#action_content"
func GetRolesByAcctname(c *gin.Context) {
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

	// get working acct id
	var workingAccountId int64

	resp1 := getAccountByName(client, mctx, session.Get("working_actname").(string))
	workingAccountId = resp1.GetAccount().AccountId

	// get account roles
	resp := getAccountRoles(client, mctx, workingAccountId)

	c.HTML(http.StatusOK,
		"role_list.html",
		gin.H{
			"auth":               c.GetBool("auth"),
			"claims":             c.GetStringMap("claims"),
			"roles":              resp,
			"acctname":           session.Get("working_actname").(string),
			"working_account_id": workingAccountId,
		},
	)

}

// Get Role Form - hx-target="#action_content"
func GetAccountRoleById(c *gin.Context) {
	// session := sessions.Default(c)
	var working_account_id int64

	fmt.Println("raw role id:", c.Param("id"))

	if c.Param("id") != "new" {
		roleId, _ := strconv.ParseInt(c.Param("id"), 10, 64)

		conn := getGrpcConn()
		defer conn.Close()

		client := pb.NewMServiceAccountClient(conn)

		mctx := getMctxFromCookie(c)

		resp := getAccountRoleById(client, mctx, roleId)
		c.HTML(http.StatusOK,
			"role_form.html",
			gin.H{
				"auth":   c.GetBool("auth"),
				"claims": c.GetStringMap("claims"),
				"role":   resp,
			},
		)

	} else { // new role rec
		fmt.Println("raw acctid param............", c.Request.URL.Query()["acctid"][0])

		i, err := strconv.ParseInt(c.Request.URL.Query()["acctid"][0], 10, 64)
		if err != nil {
			panic(err)
		} else {
			working_account_id = i
		}
		fmt.Println("working acctid for new:::", working_account_id)

		c.HTML(http.StatusOK,
			"role_form.html",
			gin.H{
				"auth":               c.GetBool("auth"),
				"claims":             c.GetStringMap("claims"),
				"isCreate":           true,
				"working_account_id": working_account_id,
			},
		)

	}

}

// Update role and redisplay role form - hx-target="#action_content"
func UpdateRole(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("update current working acct:", session.Get("working_actname"))

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.UpdateAccountRoleRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// fmt.Println("req ????", req)
	// fmt.Println("new account name::::::", req.AccountName)

	resp, err := client.UpdateAccountRole(mctx, &req)
	// fmt.Println(resp)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update account error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"role_form.html",
			gin.H{
				"user":    req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
			},
		)

	} else {
		// reget the role by id
		resp1 := getAccountRoleById(client, mctx, req.RoleId)

		c.HTML(http.StatusOK,
			"role_form.html",
			gin.H{
				"auth":    c.GetBool("auth"),
				"claims":  c.GetStringMap("claims"),
				"role":    resp1,
				"success": true,
			},
		)
	}

}

// Create new Role and display Role form - hx-target="#action_content"
func CreateRole(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("update current working acct:", session.Get("working_actname"))

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.CreateAccountRoleRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// fmt.Println("req ????", req)
	fmt.Println("new account name::::::", req.RoleName)

	resp, err := client.CreateAccountRole(mctx, &req)
	// fmt.Println(resp)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update role error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"role_form.html",
			gin.H{
				"working_account_id": req.AccountId,
				"role_name":          req.RoleName,
				"failure":            true,
				"claims":             c.GetStringMap("claims"),
				"isCreate":           true,
			},
		)

	} else {
		// get the role by id
		resp1 := getAccountRoleById(client, mctx, resp.RoleId)

		c.HTML(http.StatusOK,
			"role_form.html",
			gin.H{
				"auth":    c.GetBool("auth"),
				"claims":  c.GetStringMap("claims"),
				"role":    resp1,
				"success": true,
			},
		)
	}

}

func DeleteRole(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("delete role current working acct:", session.Get("working_actname"))

	roleId, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	// var version int32
	version, _ := strconv.ParseInt(c.Param("version"), 10, 32)

	// acctname := session.Get("actname").(string)

	fmt.Println("id:", roleId)
	// fmt.Println("version:", version)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// Delete the record
	req := pb.DeleteAccountRoleRequest{}
	req.RoleId = roleId
	req.Version = int32(version)
	resp, err := client.DeleteAccountRole(mctx, &req)
	if err == nil {
		jtext, err := json.MarshalIndent(resp, "", "  ")
		if err == nil {
			fmt.Println(string(jtext))
		}
	}

	// // if ok set working_actname to actname get account names
	// session.Set("working_actname", acctname)
	var acctnames []string

	if session.Get("actname") == "master" {
		resp1 := getAccountNames(client, mctx)
		acctnames = resp1.GetAccountName()
	}

	fmt.Println("delete before render", session.Get("working_actname"))

	// get working acct id
	var workingAccountId int64

	resp2 := getAccountByName(client, mctx, session.Get("working_actname").(string))
	workingAccountId = resp2.GetAccount().AccountId

	// then get working account roles
	resp1 := getAccountRoles(client, mctx, workingAccountId)

	c.HTML(http.StatusOK,
		"roles.html",
		gin.H{
			"auth":               c.GetBool("auth"),
			"claims":             c.GetStringMap("claims"),
			"roles":              resp1,
			"acctname":           session.Get("working_actname").(string),
			"working_account_id": workingAccountId,
			"acctnames":          acctnames,
		},
	)

}

// Get Role claim assignment Form - hx-target="#action_content"
func GetAssignClaimsToRoleForm(c *gin.Context) {
	// session := sessions.Default(c)

	fmt.Println("raw role id:", c.Param("id"))

	roleId, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// first get the role record w/assigned claims
	resp := getAccountRoleById(client, mctx, roleId)

	// next get all the available claim / claim values
	resp1 := getClaimNames(client, mctx)

	//  now compare available vs assigned
	available_claims := getAvailableClaims(*resp.AccountRole, resp1)

	unique_available_claims := lo.Uniq(available_claims)

	// fmt.Println("unique_available_claims", unique_available_claims)

	// now get claim values for the unique available_claims
	var combined_claim_values []pb.ClaimValue
	for _, claimId := range unique_available_claims {
		resp := getClaimValuesByNameId(client, claimId, mctx)
		for _, claimValue := range resp.ClaimValue {
			combined_claim_values = append(combined_claim_values, *claimValue)
		}
	}

	c.HTML(http.StatusOK,
		"role_claim_assignment_form.html",
		gin.H{
			"auth":      c.GetBool("auth"),
			"claims":    c.GetStringMap("claims"),
			"role":      resp,
			"available": combined_claim_values,
		},
	)

}

// add a claim to role then reGet Role claim assignment Form - hx-target="#action_content"
func AddClaimToRole(c *gin.Context) {
	// session := sessions.Default(c)
	fmt.Println("********************* add claim to role?")

	roleId, _ := strconv.ParseInt(c.Param("role_id"), 10, 64)
	// var version int32
	claimValId, _ := strconv.ParseInt(c.Param("claimval_id"), 10, 32)

	// cctname := session.Get("actname").(string)

	fmt.Println("id:", roleId)
	fmt.Println("claimValId:", claimValId)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// add the claim
	req0 := pb.AddClaimToRoleRequest{}
	req0.RoleId = roleId
	req0.ClaimValueId = claimValId
	resp0, err0 := client.AddClaimToRole(mctx, &req0)
	if err0 == nil {
		jtext, err := json.MarshalIndent(resp0, "", "  ")
		if err == nil {
			fmt.Println("resp from add claim********")
			fmt.Println(string(jtext))
		}
	}

	// now reget the role and available claims and redisplay the form
	resp := getAccountRoleById(client, mctx, roleId)

	// next get all the available claim / claim values
	resp1 := getClaimNames(client, mctx)

	//  now compare available vs assigned
	available_claims := getAvailableClaims(*resp.AccountRole, resp1)

	unique_available_claims := lo.Uniq(available_claims)

	// fmt.Println("unique_available_claims", unique_available_claims)

	// now get claim values for the unique available_claims
	var combined_claim_values []pb.ClaimValue
	for _, claimId := range unique_available_claims {
		resp := getClaimValuesByNameId(client, claimId, mctx)
		for _, claimValue := range resp.ClaimValue {
			combined_claim_values = append(combined_claim_values, *claimValue)
		}
	}

	c.HTML(http.StatusOK,
		"role_claim_assignment_form.html",
		gin.H{
			"auth":      c.GetBool("auth"),
			"claims":    c.GetStringMap("claims"),
			"role":      resp,
			"available": combined_claim_values,
			"success":   true,
		},
	)

}

// add a claim to role then reGet Role claim assignment Form - hx-target="#action_content"
func RemoveClaimFromRole(c *gin.Context) {
	// session := sessions.Default(c)
	fmt.Println("********************* remove claim from role?")

	roleId, _ := strconv.ParseInt(c.Param("role_id"), 10, 64)
	// var version int32
	claimValId, _ := strconv.ParseInt(c.Param("claimval_id"), 10, 32)

	// cctname := session.Get("actname").(string)

	fmt.Println("id:", roleId)
	fmt.Println("claimValId:", claimValId)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// add the claim
	req0 := pb.RemoveClaimFromRoleRequest{}
	req0.RoleId = roleId
	req0.ClaimValueId = claimValId
	resp0, err0 := client.RemoveClaimFromRole(mctx, &req0)
	if err0 == nil {
		jtext, err := json.MarshalIndent(resp0, "", "  ")
		if err == nil {
			fmt.Println("resp from remove claim********")
			fmt.Println(string(jtext))
		}
	}

	// now reget the role and available claims and redisplay the form
	// first get the role record w/assigned claims
	resp := getAccountRoleById(client, mctx, roleId)

	// next get all the available claim / claim values
	resp1 := getClaimNames(client, mctx)

	//  now compare available vs assigned
	available_claims := getAvailableClaims(*resp.AccountRole, resp1)

	unique_available_claims := lo.Uniq(available_claims)

	// fmt.Println("unique_available_claims", unique_available_claims)

	// now get claim values for the unique available_claims
	var combined_claim_values []pb.ClaimValue
	for _, claimId := range unique_available_claims {
		resp := getClaimValuesByNameId(client, claimId, mctx)
		for _, claimValue := range resp.ClaimValue {
			combined_claim_values = append(combined_claim_values, *claimValue)
		}
	}

	c.HTML(http.StatusOK,
		"role_claim_assignment_form.html",
		gin.H{
			"auth":      c.GetBool("auth"),
			"claims":    c.GetStringMap("claims"),
			"role":      resp,
			"available": combined_claim_values,
			"success":   true,
		},
	)

}
