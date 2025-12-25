package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/server/models"
)

// handleLogin authenticates a user and returns a JWT token
// @Summary      Login
// @Description  Authenticates a user with username and password and returns a JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body models.LoginRequest true "Login credentials"
// @Success      200 {object} models.LoginResponse "Login successful"
// @Failure      401 {object} models.ErrorResponse "Unauthorized - invalid credentials"
// @Failure      400 {object} models.ErrorResponse "Bad request - invalid input"
// @Failure      500 {object} models.ErrorResponse "Internal server error"
// @Router       /login [post]
func (h *Handler) handleLogin(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	var user models.User
	if err := h.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid username or password"})
		return
	}

	if user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "Invalid username or password"})
		return
	}

	token, err := h.GenerateToken(user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{Token: token})
}
