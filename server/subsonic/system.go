package subsonic

import (
	"net/http"
	"time"

	"github.com/stkevintan/miko/models"
)

func (s *Subsonic) handlePing(w http.ResponseWriter, r *http.Request) {
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Ping = &models.Ping{}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetLicense(w http.ResponseWriter, r *http.Request) {
	expires := time.Now().AddDate(10, 0, 0)
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.License = &models.License{
		Valid:          true,
		Email:          "miko@example.com",
		LicenseExpires: &expires,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetOpenSubsonicExtensions(w http.ResponseWriter, r *http.Request) {
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.OpenSubsonicExtensions = []models.OpenSubsonicExtension{
		{Name: "songLyrics", Versions: []int{1}},
	}
	s.sendResponse(w, r, resp)
}
