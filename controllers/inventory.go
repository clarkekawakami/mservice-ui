package other_controllers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func GetInventoryPage(c *gin.Context) {
	session := sessions.Default(c)

	c.HTML(http.StatusOK,
		"app_page.html",
		gin.H{
			"title":      "Inventory Management",
			"active":     "inventory",
			"sub_active": "two",
			"sub_title":  "Inventory Mgmt Stuff",
			"auth":       c.GetBool("auth"),
			"claims":     c.GetStringMap("claims"),
			"fullname":   session.Get("fullname"),
		},
	)

}
