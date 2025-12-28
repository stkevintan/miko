package subsonic

import (
	"net/http"
	"strconv"
	"time"

	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) handleStar(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ids := query["id"]
	albumIds := query["albumId"]
	artistIds := query["artistId"]

	db := do.MustInvoke[*gorm.DB](s.injector)
	now := time.Now()

	if len(ids) > 0 {
		db.Model(&models.Child{}).Where("id IN ?", ids).Update("starred", &now)
	}
	if len(albumIds) > 0 {
		db.Model(&models.AlbumID3{}).Where("id IN ?", albumIds).Update("starred", &now)
	}
	if len(artistIds) > 0 {
		db.Model(&models.ArtistID3{}).Where("id IN ?", artistIds).Update("starred", &now)
	}

	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}

func (s *Subsonic) handleUnstar(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ids := query["id"]
	albumIds := query["albumId"]
	artistIds := query["artistId"]

	db := do.MustInvoke[*gorm.DB](s.injector)

	if len(ids) > 0 {
		db.Model(&models.Child{}).Where("id IN ?", ids).Update("starred", nil)
	}
	if len(albumIds) > 0 {
		db.Model(&models.AlbumID3{}).Where("id IN ?", albumIds).Update("starred", nil)
	}
	if len(artistIds) > 0 {
		db.Model(&models.ArtistID3{}).Where("id IN ?", artistIds).Update("starred", nil)
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

	db := do.MustInvoke[*gorm.DB](s.injector)

	// Try to update as song/directory first
	result := db.Model(&models.Child{}).Where("id = ?", id).Update("user_rating", rating)
	if result.RowsAffected == 0 {
		// Try to update as album
		result = db.Model(&models.AlbumID3{}).Where("id = ?", id).Update("user_rating", rating)
		if result.RowsAffected == 0 {
			// Try to update as artist
			db.Model(&models.ArtistID3{}).Where("id = ?", id).Update("user_rating", rating)
		}
	}

	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}
