package subsonic

import (
	"crypto/md5"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/api/models"
	"gorm.io/gorm"
)

type Subsonic struct {
	db   *gorm.DB
	rest *gin.RouterGroup
}

func NewSubsonic(r *gin.Engine, db *gorm.DB) *Subsonic {
	return &Subsonic{
		db:   db,
		rest: r.Group("/rest"),
	}
}

func (s *Subsonic) RegisterRoutes() {
	s.rest.Use(s.subsonicAuthMiddleware())
	s.rest.GET("/ping.view", s.handlePing)
}

func (s *Subsonic) sendResponse(c *gin.Context, resp *SubsonicResponse) {
	format := c.DefaultQuery("f", "xml")
	if format == "json" {
		c.JSON(http.StatusOK, gin.H{"subsonic-response": resp})
	} else {
		c.XML(http.StatusOK, resp)
	}
}

func (s *Subsonic) subsonicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Query("u")
		password := c.Query("p")
		token := c.Query("t")
		salt := c.Query("s")

		if username == "" {
			s.sendResponse(c, NewErrorResponse(10, "User not found"))
			c.Abort()
			return
		}

		var user models.User
		if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
			s.sendResponse(c, NewErrorResponse(10, "User not found"))
			c.Abort()
			return
		}

		authenticated := false
		if password != "" {
			// Clear text password auth
			if user.Password == password {
				authenticated = true
			}
		} else if token != "" && salt != "" {
			// Token auth: t = md5(password + salt)
			expectedToken := fmt.Sprintf("%x", md5.Sum([]byte(user.Password+salt)))
			if expectedToken == token {
				authenticated = true
			}
		}

		if !authenticated {
			s.sendResponse(c, NewErrorResponse(40, "Wrong username or password"))
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
