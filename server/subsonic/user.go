package subsonic

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) handleGetUser(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "Username is required"))
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)
	var user models.User
	if err := db.Preload("SubsonicSettings").Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.sendResponse(c, models.NewErrorResponse(70, "User not found"))
		} else {
			s.sendResponse(c, models.NewErrorResponse(0, "An internal error occurred"))
		}
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.User = &user
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetUsers(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	var users []models.User
	if err := db.Preload("SubsonicSettings").Find(&users).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.sendResponse(c, models.NewErrorResponse(70, "User not found"))
		} else {
			s.sendResponse(c, models.NewErrorResponse(0, "An internal error occurred"))
		}
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Users = &models.Users{
		User: users,
	}
	s.sendResponse(c, resp)
}
