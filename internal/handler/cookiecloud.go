package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/internal/models"
)

// handleCookiecloudIdentity updates CookieCloud identity
// @Summary      Update CookieCloud identity
// @Description  Updates the CookieCloud server identity using the provided key and password. This allows the service to fetch cookies from your CookieCloud server for authentication with music platforms.
// @Tags         cookiecloud
// @Accept       json
// @Produce      json
// @Param        body body models.CookiecloudIdentity true "CookieCloud identity (key and password)"
// @Success      200 {object} models.CookiecloudIdentityResponse "Identity updated successfully"
// @Failure      400 {object} models.ErrorResponse "Bad request - invalid JSON or missing required fields"
// @Failure      500 {object} models.ErrorResponse "Internal server error - failed to update identity"
// @Router       /cookiecloud/identity [post]
func (h *Handler) handleCookiecloudIdentity(c *gin.Context) {

	var req models.CookiecloudIdentity
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		c.JSON(http.StatusBadRequest, errorResp)
		return
	}
	err := h.jar.UpdateIdentity(req.Key, req.Password)

	if err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		c.JSON(http.StatusInternalServerError, errorResp)
		return
	}

	response := models.CookiecloudIdentityResponse{
		Url: h.jar.GetUrl(),
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
// @Router       /cookiecloud/server [get]
func (h *Handler) getCookiecloudServer(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"url": h.jar.GetUrl(),
	})
}
