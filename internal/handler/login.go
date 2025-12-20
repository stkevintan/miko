package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/internal/models"
)

// handleLogin handles user login
// @Summary      User login
// @Description  Authenticates user with provided credentials
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        platform query string false "Music platform" example("netease")
// @Param        body body models.LoginRequest true "Login request"
// @Success      200 {object} models.LoginResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /login [post]
func (h *Handler) handleLogin(c *gin.Context) {
	platform := c.DefaultQuery("platform", h.registry.Config.Platform)

	var req models.LoginRequest
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

	result, err := provider.Login(c.Request.Context(), req.UUID, req.Password)

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

	response := models.LoginResponse{
		Username: result.Username,
		UserID:   result.UserID,
		Success:  true,
		Message:  "Login successful",
	}

	c.JSON(http.StatusOK, response)
}
