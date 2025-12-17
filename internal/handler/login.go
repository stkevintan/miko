package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/internal/models"
	"github.com/stkevintan/miko/internal/service"
)

// handleLogin handles user login
// @Summary      User login
// @Description  Authenticates user with provided credentials
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.LoginRequest true "Login credentials"
// @Success      200 {object} models.LoginResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /login [post]
func (h *Handler) handleLogin(c *gin.Context) {
	var req models.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		errorResp := models.ErrorResponse{Error: "Invalid JSON: " + err.Error()}
		c.JSON(http.StatusBadRequest, errorResp)
		return
	}

	// Convert timeout from milliseconds to duration
	timeout := time.Duration(req.Timeout) * time.Millisecond
	if timeout == 0 {
		timeout = 30 * time.Second // default timeout
	}

	result, err := h.service.Login(c.Request.Context(), &service.LoginArgs{
		Timeout:  timeout,
		Server:   req.Server,
		UUID:     req.UUID,
		Password: req.Password,
	})

	if err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		c.JSON(http.StatusInternalServerError, errorResp)
		return
	}

	if result == nil {
		errorResp := models.ErrorResponse{Error: "Login failed: no result returned"}
		c.JSON(http.StatusInternalServerError, errorResp)
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
