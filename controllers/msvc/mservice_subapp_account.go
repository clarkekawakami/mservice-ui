package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	pb "github.com/gaterace/mservice/pkg/mserviceaccount"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// *********************
// local funcs
// *********************

func f64ToI64(f float64) int64 {
	var i int64 = int64(f)
	return i

}

func getGrpcConn() *grpc.ClientConn {
	fmt.Println("in getGrpcConn")
	sAddress := "localhost:50051"
	conn, e := grpc.Dial(sAddress, grpc.WithInsecure())
	if e != nil {
		log.Fatalf("Failed to connect to server %v", e)
	}
	return conn
}

func getMctxFromCookie(c *gin.Context) context.Context {
	fmt.Println("in getMctxFromCookie")
	token, err := c.Cookie("token")

	if err != nil {
		token = "NotSet"
		c.SetCookie("token", token, 3600, "/", "localhost", false, true)
	}

	md := metadata.Pairs("token", token)

	return metadata.NewOutgoingContext(context.Background(), md)
}

// *********************************
// local funcs to get mservice data
// *********************************

func getAccountUserById(client pb.MServiceAccountClient, id int64, mctx context.Context) *pb.GetAccountUserByIdResponse {
	fmt.Println("in getAccountUserById func")
	req := pb.GetAccountUserByIdRequest{}
	req.UserId = id
	resp, err := client.GetAccountUserById(mctx, &req)

	if err != nil {
		fmt.Printf("err: %s\n", err)
	}

	return resp

}

func getAccountNames(client pb.MServiceAccountClient, mctx context.Context) *pb.GetAccountNamesResponse {
	fmt.Println("in getAccountNames func")
	req := pb.GetAccountNamesRequest{}
	req.DummyParam = 1
	resp, err := client.GetAccountNames(mctx, &req)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	return resp
}

func getAccountByName(client pb.MServiceAccountClient, mctx context.Context, acctname string) *pb.GetAccountByNameResponse {
	fmt.Println("in getAccountByName func")
	req := pb.GetAccountByNameRequest{}
	req.AccountName = acctname
	resp, err := client.GetAccountByName(mctx, &req)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	return resp
}

// ***********************************************
// public funcs to send back htmx fragments/pages
// ***********************************************

// login to msvc server and render logged-in page
// since only 1 func uses login put login request trans here
func Login(c *gin.Context) {
	session := sessions.Default(c)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	req := pb.LoginRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var tokData string

	resp, err := client.Login(context.Background(), &req)

	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("login error:", resp.ErrorCode, resp.ErrorMessage)

		// login failed
		c.HTML(http.StatusOK,
			"loggedin_page.html",
			gin.H{
				"title": "Login Failed",
				"auth":  false,
			},
		)

	} else { // login success... set session vars... user full name requires us to get the user record
		tokData = resp.GetJwt()

		c.SetCookie("token", tokData, 36000, "/", "localhost", false, true)

		// decode jwt token

		token, err := jwt.Parse(string(tokData), nil)
		if token == nil {
			fmt.Println("malformed token: ", err)
		}

		var out []byte
		var err1 error
		var fullname string
		var t_iss time.Time
		var t_exp time.Time

		var claims map[string]interface{}

		out, err1 = json.Marshal(token.Claims)

		if err1 == nil {
			json.Unmarshal([]byte(out), &claims)
			// fmt.Println("claims map?", claims)
			// session.Set("acctmgt", claims["acctmgt"])
			session.Set("actname", claims["actname"])
			session.Set("working_actname", claims["actname"])
			session.Set("aid", f64ToI64(claims["aid"].(float64)))
			session.Set("exp", f64ToI64(claims["exp"].(float64)))
			// session.Set("invsvc", claims["invsvc"])
			// session.Set("iss", f64ToI64(claims["iss"].(float64)))
			// session.Set("projsvc", claims["projsvc"])
			// session.Set("safebox", claims["safebox"])
			session.Set("uid", f64ToI64(claims["uid"].(float64)))

			// get user's full name
			md := metadata.Pairs("token", tokData)

			mctx := metadata.NewOutgoingContext(context.Background(), md)

			user_id := f64ToI64(claims["uid"].(float64))

			accountUser := getAccountUserById(client, user_id, mctx)

			fullname = accountUser.AccountUser.UserFullName
			fmt.Println("user full name??????", fullname)

			session.Set("email", accountUser.AccountUser.Email)
			session.Set("fullname", accountUser.AccountUser.UserFullName)
			session.Save()

			t_iss = time.Unix(f64ToI64(claims["iss"].(float64)), 0)
			t_exp = time.Unix(session.Get("exp").(int64), 0)

		}

		c.HTML(http.StatusOK,
			"loggedin_page.html",
			gin.H{
				"title":    "Login Successful",
				"auth":     true,
				"claims":   claims,
				"fullname": fullname,
				"exp":      t_exp.Format("01/02/2006, 15:04:05"),
				"iss":      t_iss.Format("01/02/2006, 15:04:05"),
			},
		)
	}
}

// clear session vars and token cookie and render home page
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear() // this will mark the session as "written" only if there's
	// at least one key to delete
	session.Options(sessions.Options{MaxAge: -1})
	session.Save()
	c.SetCookie("token", "NotSet", 36000, "/", "localhost", false, true)

	c.HTML(http.StatusOK,
		"index.html",
		gin.H{
			"title": "Logout Successful",
			"auth":  false,
		},
	)

}

// get Account data for msvc/account... target sub_app_content
func GetSubappAccount(c *gin.Context) {
	// fmt.Println("in getAccounts1")
	session := sessions.Default(c)
	fmt.Println("your account is :", session.Get("actname"))
	if session.Get("working_actname") == nil {
		session.Set("working_actname", session.Get("actname"))
		session.Save()
	}
	fmt.Println("working account is :", session.Get("working_actname"))

	acctname := session.Get("actname").(string)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	var acctnames []string
	if session.Get("actname") == "master" {

		resp := getAccountNames(client, mctx)
		acctnames = resp.GetAccountName()

	}
	// fmt.Println("last", acctnames)

	if acctname != "new" {
		resp := getAccountByName(client, mctx, acctname)

		// fmt.Println("claims???????????", c.GetStringMap("claims"))
		c.HTML(http.StatusOK,
			"subapp_partial.html",
			gin.H{
				"sub_title":  "MService Account Details",
				"sub_active": "account",
				"acctnames":  acctnames,
				"acct_name":  acctname,
				"acct":       resp,
				"auth":       c.GetBool("auth"),
				"claims":     c.GetStringMap("claims"),
			},
		)

	} else {
		c.HTML(http.StatusOK,
			"subapp_partial.html",
			gin.H{
				"sub_title":  "MService Account Details",
				"sub_active": "account",
				"auth":       c.GetBool("auth"),
				"claims":     c.GetStringMap("claims"),
			},
		)
	}

}

// get a different Account by Name target #sub_app_content
func GetAcctByName(c *gin.Context) {
	session := sessions.Default(c)
	// get query param
	acctname := c.Request.URL.Query()["acctname"][0]
	fmt.Println("****working account*****", acctname)
	session.Set("working_actname", acctname)
	session.Save()

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	var acctnames []string
	if session.Get("actname") == "master" {

		resp := getAccountNames(client, mctx)
		acctnames = resp.GetAccountName()

	}

	if acctname != "new" {
		resp := getAccountByName(client, mctx, acctname)

		// fmt.Println("claims???????????", c.GetStringMap("claims"))
		c.HTML(http.StatusOK,
			"subapp_partial.html",
			gin.H{
				"sub_title":  "MService Account Details",
				"sub_active": "account",
				"acctnames":  acctnames,
				"acct_name":  acctname,
				"acct":       resp,
				"auth":       c.GetBool("auth"),
				"claims":     c.GetStringMap("claims"),
			},
		)

	} else {
		c.HTML(http.StatusOK,
			"subapp_partial.html",
			gin.H{
				"sub_title":  "MService Account Details",
				"sub_active": "account",
				"acctnames":  acctnames,
				"acct_name":  acctname,
				"auth":       c.GetBool("auth"),
				"claims":     c.GetStringMap("claims"),
				"isCreate":   true,
			},
		)
	}
}

// Get msvc server info/uptime - hx-target="#sub_app_content"
func GetServerVersion(c *gin.Context) {

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	// version := getServerVersion(client)
	req := pb.GetServerVersionRequest{}
	req.DummyParam = 1
	version, err := client.GetServerVersion(context.Background(), &req)
	// if err == nil {
	// 	jtext, err := json.MarshalIndent(resp, "", "  ")
	// 	if err == nil {
	// 		fmt.Println(string(jtext))
	// 	}
	// }

	if err != nil {
		fmt.Printf("err: %s\n", err)
	}

	uptime_hours := version.GetServerUptime() / 3600
	uptime_minutes := (version.GetServerUptime() % 3600) / 60
	uptime_seconds := (version.GetServerUptime() % 60)

	fmt.Println("hours", uptime_hours)
	fmt.Println("minutes", uptime_minutes)
	fmt.Println("seconds", uptime_seconds)

	var uptime_string string

	uptime_string = "Uptime: " + strconv.FormatInt(uptime_hours, 10) + " hours " + strconv.FormatInt(uptime_minutes, 10) + " minutes " + strconv.FormatInt(uptime_seconds, 10) + " seconds"
	fmt.Println(uptime_string)

	c.HTML(http.StatusOK,
		"subapp_partial.html",
		gin.H{
			"sub_title":  "MService Server Status",
			"sub_active": "server",
			// "acctnames":  acctnames,
			// "acct_name":  acctname,
			"version": version.ServerVersion,
			"uptime":  uptime_string,
			"auth":    c.GetBool("auth"),
			"claims":  c.GetStringMap("claims"),
		},
	)

}

// update account record
func UpdateAccount(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("update current working acct:", session.Get("working_actname"))

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.UpdateAccountRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// fmt.Println("req ????", req)
	// fmt.Println("new account name::::::", req.AccountName)

	resp, err := client.UpdateAccount(mctx, &req)
	// fmt.Println(resp)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update account error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"acct_form.html",
			gin.H{
				"acct":    req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
			},
		)

	} else {
		// get acctnames
		var acctnames []string

		if session.Get("actname") == "master" {

			resp := getAccountNames(client, mctx)
			acctnames = resp.GetAccountName()

		}
		// reget the account by name

		resp1 := getAccountByName(client, mctx, req.AccountName)

		c.HTML(http.StatusOK,
			"subapp_partial.html",
			gin.H{
				"sub_title":  "MService Account Details",
				"sub_active": "account",
				"acctnames":  acctnames,
				"acct_name":  resp1.Account.AccountName,
				"auth":       c.GetBool("auth"),
				"acct":       resp1,
				"success":    true,
				"claims":     c.GetStringMap("claims"),
			},
		)

	}

}

func CreateAccount(c *gin.Context) {
	session := sessions.Default(c)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	req := pb.CreateAccountRequest{}

	// This will infer what binder to use depending on the content-type header.
	if err := c.ShouldBind(&req); err != nil {
		fmt.Println("bind error:::::::::", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// fmt.Println("req ????", req)
	fmt.Println("new account name::::::", req.AccountName)

	resp, err := client.CreateAccount(mctx, &req)
	// fmt.Println(resp)
	if err != nil {
		fmt.Printf("err: %s\n", err)
	}
	if resp.GetErrorCode() != 0 {
		fmt.Println("update account error:", resp.ErrorCode, resp.ErrorMessage)

		// update account failed
		c.HTML(http.StatusOK,
			"acct_form.html",
			gin.H{
				"acct":    req,
				"failure": true,
				"claims":  c.GetStringMap("claims"),
			},
		)

	} else {
		// get acctnames
		var acctnames []string

		if session.Get("actname") == "master" {
			resp := getAccountNames(client, mctx)
			acctnames = resp.GetAccountName()
		}

		// reget the account by name

		resp1 := getAccountByName(client, mctx, req.AccountName)
		session.Set("working_actname", req.AccountName)
		session.Save()

		c.HTML(http.StatusOK,
			"subapp_partial.html",
			gin.H{
				"sub_title":  "MService Account Details",
				"sub_active": "account",
				"acctnames":  acctnames,
				"acct_name":  resp1.Account.AccountName,
				"auth":       c.GetBool("auth"),
				"acct":       resp1,
				"success":    true,
				"claims":     c.GetStringMap("claims"),
			},
		)

	}

}

func DeleteAccount(c *gin.Context) {
	session := sessions.Default(c)
	fmt.Println("delete current working acct:", session.Get("working_actname"))

	accountId, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	// var version int32
	version, _ := strconv.ParseInt(c.Param("version"), 10, 32)

	acctname := session.Get("actname").(string)

	fmt.Println("id:", accountId)
	// fmt.Println("version:", version)

	conn := getGrpcConn()
	defer conn.Close()

	client := pb.NewMServiceAccountClient(conn)

	mctx := getMctxFromCookie(c)

	// Delete the record
	req := pb.DeleteAccountRequest{}
	req.AccountId = accountId
	req.Version = int32(version)
	resp, err := client.DeleteAccount(mctx, &req)
	if err == nil {
		jtext, err := json.MarshalIndent(resp, "", "  ")
		if err == nil {
			fmt.Println(string(jtext))
		}
	}

	// if ok set working_actname to actname get account names
	session.Set("working_actname", acctname)
	session.Save()
	var acctnames []string
	if session.Get("actname") == "master" {
		resp := getAccountNames(client, mctx)
		acctnames = resp.GetAccountName()
	}

	// then get master account record

	resp1 := getAccountByName(client, mctx, acctname)

	fmt.Println("delete before render", session.Get("working_actname"))

	c.HTML(http.StatusOK,
		"account.html",
		gin.H{
			"acctnames": acctnames,
			"acct_name": acctname,
			"acct":      resp1,
			"auth":      c.GetBool("auth"),
			"claims":    c.GetStringMap("claims"),
		},
	)

}
