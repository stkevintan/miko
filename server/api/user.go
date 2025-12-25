package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/pkg/registry"
	"github.com/stkevintan/miko/server/models"
)

// handleUser retrieves user information from a music platform
// @Summary      Get user information
// @Description  Retrieves the authenticated user's information from the specified music platform
// @Tags         user
// @Produce      json
// @Param        platform path string true "Music platform name" example("netease")
// @Success      200 {object} types.User "User information retrieved successfully"
// @Failure      500 {object} models.ErrorResponse "Internal server error - authentication failed or invalid platform"
// @Failure      400 {object} models.ErrorResponse "Bad request - provider creation failed"
// @Security     ApiKeyAuth
// @Router       /platform/{platform}/user [get]
func (h *Handler) handlePlatformUser(c *gin.Context) {
	platform := c.Param("platform")

	injector, err := h.getRequestInjector(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}

	provider, err := do.InvokeNamed[registry.Provider](injector, platform)

	if err != nil {
		c.JSON(http.StatusBadRequest, &models.ErrorResponse{Error: err.Error()})
		return
	}
	user, err := provider.User(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, user)
}

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
