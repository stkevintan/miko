package subsonic

import (
	"errors"
	"net/http"

	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) handleGetUser(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "Username is required"))
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)
	var user models.User
	if err := db.Preload("MusicFolders").Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.sendResponse(w, r, models.NewErrorResponse(70, "User not found"))
		} else {
			s.sendResponse(w, r, models.NewErrorResponse(0, "An internal error occurred"))
		}
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.User = &user
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	var users []*models.User
	if err := db.Preload("MusicFolders").Find(&users).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "An internal error occurred"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Users = &models.Users{
		User: users,
	}
	s.sendResponse(w, r, resp)
}
