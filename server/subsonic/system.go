package subsonic

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/models"
)

func (s *Subsonic) handlePing(c *gin.Context) {
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Ping = &models.Ping{}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetLicense(c *gin.Context) {
	expires := time.Now().AddDate(10, 0, 0)
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.License = &models.License{
		Valid:          true,
		Email:          "miko@example.com",
		LicenseExpires: &expires,
	}
	s.sendResponse(c, resp)
}
