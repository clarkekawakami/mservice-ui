package other_controllers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func GetProjectPage(c *gin.Context) {
	session := sessions.Default(c)

	c.HTML(http.StatusOK,
		"app_page.html",
		gin.H{
			"title":      "Project Management",
			"active":     "project",
			"sub_active": "one",
			"sub_title":  "Project Management Stuff",
			"auth":       c.GetBool("auth"),
			"claims":     c.GetStringMap("claims"),
			"fullname":   session.Get("fullname"),
		},
	)

}
