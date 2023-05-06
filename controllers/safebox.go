package other_controllers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func GetSafeboxPage(c *gin.Context) {
	session := sessions.Default(c)

	c.HTML(http.StatusOK,
		"app_page.html",
		gin.H{
			"title":      "Safebox",
			"active":     "safebox",
			"sub_active": "three",
			"sub_title":  "Safebox Stuff",
			"auth":       c.GetBool("auth"),
			"claims":     c.GetStringMap("claims"),
			"fullname":   session.Get("fullname"),
		},
	)

}
