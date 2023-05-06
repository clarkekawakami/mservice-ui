package other_controllers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func GetLedgerPage(c *gin.Context) {
	session := sessions.Default(c)

	c.HTML(http.StatusOK,
		"app_page.html",
		gin.H{
			"title":      "General Ledger",
			"active":     "ledger",
			"sub_active": "three",
			"sub_title":  "General Ledger Stuff",
			"auth":       c.GetBool("auth"),
			"claims":     c.GetStringMap("claims"),
			"fullname":   session.Get("fullname"),
		},
	)

}
