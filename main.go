package main

import (
	"encoding/json"
	"fmt"

	// "gr-ui/controllers"
	other_controllers "gr-ui/controllers"
	controllers "gr-ui/controllers/msvc"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func main() {
	r := setupRouter()
	_ = r.Run(":3000")
}

func checkSession(c *gin.Context) {
	session := sessions.Default(c)

	fmt.Println("chk session current working acct:", session.Get("working_actname"))
	// if err == nil {
	// 	fmt.Println("in err == nil... tokData:", reflect.TypeOf(tokData), tokData)

	current := time.Now().Unix()
	// fmt.Println("session exp:", session.Get("exp"))
	if session.Get("exp") != nil {
		if session.Get("exp").(int64) > current {
			fmt.Println("ok", (session.Get("exp").(int64)-current)/60)

			var claims map[string]interface{}
			tokData, err := c.Cookie("token")

			token, err := jwt.Parse(string(tokData), nil)
			if token == nil {
				fmt.Println("malformed token: ", err)
			} else {
				// fmt.Println("in else... token:", reflect.TypeOf(token), token)
				// now turn claims into map
				var out []byte
				var err1 error

				out, err1 = json.Marshal(token.Claims)

				if err1 == nil {
					json.Unmarshal([]byte(out), &claims)
					// fmt.Println("claims map?", claims)
				}
			}
			c.Set("claims", claims)
			c.Set("auth", true)
		} else {
			fmt.Println("expired!", (session.Get("exp").(int64)-current)/60)
			c.Set("auth", false)
			c.SetCookie("token", "NotSet", 36000, "/", "localhost", false, true)
		}
	} else {
		c.Set("auth", false)
		c.SetCookie("token", "NotSet", 36000, "/", "localhost", false, true)
	}

	// Pass on to the next-in-chain
	c.Next()
}

func setupRouter() *gin.Engine {
	r := gin.Default()

	store := cookie.NewStore([]byte("mservice claims"))
	r.Use(sessions.Sessions("mysession", store))
	r.Use(checkSession)

	r.Static("/public", "./public")
	r.StaticFile("/favicon.ico", "./public/favicon.ico")

	r.LoadHTMLGlob("templates/*")

	// app full page routes
	r.GET("/", getHomePage)
	r.GET("/login", getLoginPage)
	r.GET("/msvc", getMservicePage)
	r.GET("/msvc/me", controllers.GetMyUserRecord)
	r.GET("/msvc/change_my_password", controllers.GetChangeMyPasswordForm)
	r.POST("/msvc/change_my_password", controllers.ChangeMyPassword)

	// Mservice sub apps
	r.GET("/msvc/status", controllers.GetServerVersion)
	r.GET("/msvc/account", controllers.GetSubappAccount)
	r.GET("/msvc/users", controllers.GetSubappUsers)

	r.GET("/getusersbyacctname", controllers.GetUsersByAcctname)
	r.GET("/msvc/user/:id", controllers.GetAccountUserById)
	r.GET("/msvc/user/:id/password_reset", controllers.GetPasswordReset)
	r.GET("/acctbyname", controllers.GetAcctByName)
	r.POST("/login", controllers.Login)
	r.GET("/logout", controllers.Logout)
	r.POST("updt_acct", controllers.UpdateAccount)
	r.POST("updt_user", controllers.UpdateUser)
	r.POST("updt_claim", controllers.UpdateClaim)
	r.POST("delete_user/:id/:version", controllers.DeleteUser)
	r.POST("create_user", controllers.CreateUser)
	r.POST("create_acct", controllers.CreateAccount)
	r.POST("create_claim", controllers.CreateClaim)
	r.POST("delete_acct/:id/:version", controllers.DeleteAccount)
	r.POST("delete_claim/:id/:version", controllers.DeleteClaim)
	r.POST("reset_password", controllers.ResetPassword)
	r.GET("/msvc/claims", controllers.GetSubappClaims)
	r.GET("/msvc/claim/:id", controllers.GetClaimById)
	r.GET("/msvc/claim/values/:id", controllers.GetSubappClaimsValues)
	r.GET("/msvc/claim/value/:id", controllers.GetClaimValueById)
	r.POST("create_claim_value", controllers.CreateClaimValue)
	r.POST("updt_claim_value", controllers.UpdateClaimValue)
	r.POST("delete_claim_value/:id/:version/:claim_id", controllers.DeleteClaimValue)
	r.GET("/msvc/roles", controllers.GetSubappRoles)
	r.GET("/getrolesbyacctname", controllers.GetRolesByAcctname)
	r.GET("/msvc/role/:id", controllers.GetAccountRoleById)
	r.POST("updt_role", controllers.UpdateRole)
	r.POST("create_role", controllers.CreateRole)
	r.POST("delete_role/:id/:version", controllers.DeleteRole)
	r.GET("/msvc/role/:id/claim_assignment", controllers.GetAssignClaimsToRoleForm)
	r.POST("/add_claim_to_role/:role_id/:claimval_id", controllers.AddClaimToRole)
	r.POST("/remove_claim_from_role/:role_id/:claimval_id", controllers.RemoveClaimFromRole)
	r.GET("/msvc/user/:id/role_assignment", controllers.GetAssignRolesToUserForm)
	r.POST("/add_user_to_role/:user_id/:role_id", controllers.AddUserToRole)
	r.POST("/remove_user_from_role/:user_id/:role_id", controllers.RemoveUserFromRole)

	//Inventory routes
	r.GET("/inventory", other_controllers.GetInventoryPage)

	//Project Management routes
	r.GET("/project", other_controllers.GetProjectPage)

	//Safebox routes
	r.GET("/safebox", other_controllers.GetSafeboxPage)

	//Safebox routes
	r.GET("/ledger", other_controllers.GetLedgerPage)

	return r

}

// load full pages (no htmx)

func getHomePage(c *gin.Context) {
	session := sessions.Default(c)
	c.HTML(http.StatusOK,
		"index.html",
		gin.H{
			"title":    "Gaterace Home",
			"active":   "home",
			"auth":     c.GetBool("auth"),
			"claims":   c.GetStringMap("claims"),
			"fullname": session.Get("fullname"),
		},
	)
}

func getLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK,
		"login_form.html",
		gin.H{
			"title":  "MService Login",
			"active": "login",
			"auth":   c.GetBool("auth"),
		},
	)
}

// Load User subapp home page
func getMservicePage(c *gin.Context) {
	// fmt.Println("in getAccounts1")
	session := sessions.Default(c)
	session.Set("working_actname", session.Get("actname"))
	// fmt.Println("your account is :", session.Get("actname"))
	c.HTML(http.StatusOK,
		"app_page.html",
		gin.H{
			"title":     "MService",
			"active":    "mservice",
			"sub_title": "MService Landing Page",
			"auth":      c.GetBool("auth"),
			"claims":    c.GetStringMap("claims"),
			"fullname":  session.Get("fullname"),
		},
	)
	// }

}

// func printJSON(j interface{}) error {
// 	var out []byte
// 	var err error

// 	out, err = json.MarshalIndent(j, "", "    ")

// 	if err == nil {
// 		fmt.Println(string(out))
// 	}

// 	return err
// }
