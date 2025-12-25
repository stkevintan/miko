package api

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/pkg/provider"
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
	defer injector.Shutdown()

	provider, err := do.InvokeNamed[provider.Provider](injector, platform)

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
