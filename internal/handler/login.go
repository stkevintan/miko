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
// @Success      200 {object} models.LoginResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /login [post]
func (h *Handler) handleLogin(c *gin.Context) {
	platform := c.DefaultQuery("platform", h.registry.Config.Platform)

	provider, err := h.registry.CreateProvider(platform)
	if err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		c.JSON(http.StatusInternalServerError, errorResp)
		return
	}
	result, err := provider.Login(c.Request.Context())

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
