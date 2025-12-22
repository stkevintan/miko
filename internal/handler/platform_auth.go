package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/internal/models"
)

// handlePlatformAuth handles platform authentication
// @Summary      Platform authentication
// @Description  Authenticates a music platform using CookieCloud credentials (key + password) and returns basic user info when successful.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        platform path string true "Music platform" example("netease")
// @Param        body body models.PlatformAuthRequest true "Platform auth request"
// @Success      200 {object} models.PlatformAuthResponse "Platform auth successful"
// @Failure      400 {object} models.ErrorResponse "Bad request - invalid JSON, missing required fields, or no cookie returned"
// @Failure      500 {object} models.ErrorResponse "Internal server error - provider init/auth failure"
// @Router       /platform/{platform}/auth [post]
func (h *Handler) handlePlatformAuth(c *gin.Context) {
	platform := c.Param("platform")

	var req models.PlatformAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		c.JSON(http.StatusBadRequest, errorResp)
		return
	}

	provider, err := h.registry.CreateProvider(platform)
	if err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		c.JSON(http.StatusInternalServerError, errorResp)
		return
	}
	defer provider.Close(c.Request.Context())

	result, err := provider.Auth(c.Request.Context(), req.Key, req.Password)
	if err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		c.JSON(http.StatusInternalServerError, errorResp)
		return
	}

	if result == nil {
		errorResp := models.ErrorResponse{Error: "Login failed: no cookie from the cookiecloud"}
		c.JSON(http.StatusBadRequest, errorResp)
		return
	}

	response := models.PlatformAuthResponse{
		Username: result.Username,
		UserID:   result.UserID,
		Success:  true,
		Message:  "Login successful",
	}

	c.JSON(http.StatusOK, response)
}
