package subsonic

import (
	"net/http"
	"strconv"
	"time"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"gorm.io/gorm"
)

func (s *Subsonic) updateStarredStatus(r *http.Request, value interface{}) error {
	query := r.URL.Query()
	ids := query["id"]
	albumIds := query["albumId"]
	artistIds := query["artistId"]

	db := di.MustInvoke[*gorm.DB](s.ctx)

	return db.Transaction(func(tx *gorm.DB) error {
		if len(ids) > 0 {
			if err := tx.Model(&models.Child{}).Where("id IN ?", ids).Update("starred", value).Error; err != nil {
				return err
			}
		}
		if len(albumIds) > 0 {
			if err := tx.Model(&models.AlbumID3{}).Where("id IN ?", albumIds).Update("starred", value).Error; err != nil {
				return err
			}
		}
		if len(artistIds) > 0 {
			if err := tx.Model(&models.ArtistID3{}).Where("id IN ?", artistIds).Update("starred", value).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Subsonic) handleStar(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	if err := s.updateStarredStatus(r, &now); err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to star items: "+err.Error()))
		return
	}
	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}

func (s *Subsonic) handleUnstar(w http.ResponseWriter, r *http.Request) {
	if err := s.updateStarredStatus(r, nil); err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to unstar items: "+err.Error()))
		return
	}
	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}

func (s *Subsonic) handleSetRating(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	ratingStr := r.URL.Query().Get("rating")
	if id == "" || ratingStr == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID and rating are required"))
		return
	}

	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 0 || rating > 5 {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Invalid rating"))
		return
	}

	db := di.MustInvoke[*gorm.DB](s.ctx)

	// Try to update as song/directory first
	result := db.Model(&models.Child{}).Where("id = ?", id).Update("user_rating", rating)
	if result.Error != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to set rating: "+result.Error.Error()))
		return
	}
	if result.RowsAffected == 0 {
		// Try to update as album
		result = db.Model(&models.AlbumID3{}).Where("id = ?", id).Update("user_rating", rating)
		if result.Error != nil {
			s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to set rating: "+result.Error.Error()))
			return
		}
		if result.RowsAffected == 0 {
			// Try to update as artist
			result = db.Model(&models.ArtistID3{}).Where("id = ?", id).Update("user_rating", rating)
			if result.Error != nil {
				s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to set rating: "+result.Error.Error()))
				return
			}
			if result.RowsAffected == 0 {
				s.sendResponse(w, r, models.NewErrorResponse(70, "Item not found"))
				return
			}
		}
	}

	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}
