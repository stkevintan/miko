package subsonic

import (
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
		s.sendResponse(c, models.NewErrorResponse(10, "User not found"))
		return
	}

	// Ensure SubsonicSettings is not nil for the response
	if user.SubsonicSettings == nil {
		user.SubsonicSettings = &models.SubsonicSettings{
			StreamRole:   true,
			DownloadRole: true,
		}
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.User = &user
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetUsers(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	var users []models.User
	if err := db.Preload("SubsonicSettings").Find(&users).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to fetch users"))
		return
	}

	for i := range users {
		if users[i].SubsonicSettings == nil {
			users[i].SubsonicSettings = &models.SubsonicSettings{
				StreamRole:   true,
				DownloadRole: true,
			}
		}
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Users = &models.Users{
		User: users,
	}
	s.sendResponse(c, resp)
}
