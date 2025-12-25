package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/server/models"
)

// handleCookiecloudIdentity updates CookieCloud identity
// @Summary      Update CookieCloud identity
// @Description  Updates the CookieCloud server identity using the provided key and password. This allows the service to fetch cookies from your CookieCloud server for authentication with music platforms.
// @Tags         cookiecloud
// @Accept       json
// @Produce      json
// @Param        body body models.CookiecloudIdentityRequest true "CookieCloud identity (key and password)"
// @Success      200 {object} models.CookiecloudIdentityResponse "Identity updated successfully"
// @Failure      400 {object} models.ErrorResponse "Bad request - invalid JSON or missing required fields"
// @Failure      500 {object} models.ErrorResponse "Internal server error - failed to update identity"
// @Security     ApiKeyAuth
// @Router       /cookiecloud/identity [post]
func (h *Handler) handleCookiecloudIdentity(c *gin.Context) {

	var req models.CookiecloudIdentityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		c.JSON(http.StatusBadRequest, errorResp)
		return
	}

	username := c.MustGet("username").(string)

	// Save the identity to database associated with current account
	identity := cookiecloud.Identity{
		Username: username,
		UUID:     req.Key,
		Password: req.Password,
		URL:      h.cfg.CookieCloud.Url,
	}

	if err := h.db.Where(cookiecloud.Identity{Username: username}).
		Assign(identity).
		FirstOrCreate(&identity).Error; err != nil {
		errorResp := models.ErrorResponse{Error: "Failed to save identity: " + err.Error()}
		c.JSON(http.StatusInternalServerError, errorResp)
		return
	}

	response := models.CookiecloudIdentityResponse{
		Url: h.cfg.CookieCloud.Url,
		Key: req.Key,
	}

	c.JSON(http.StatusOK, response)
}

// getCookiecloudServer retrieves the CookieCloud server URL
// @Summary      Get CookieCloud server URL
// @Description  Returns the currently configured CookieCloud server URL
// @Tags         cookiecloud
// @Produce      json
// @Success      200 {object} map[string]string{url=string} "CookieCloud server URL"
// @Security     ApiKeyAuth
// @Router       /cookiecloud/server [get]
func (h *Handler) getCookiecloudServer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"url": h.cfg.CookieCloud.Url,
	})
}
