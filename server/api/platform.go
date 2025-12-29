package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/provider"
)

// handleUser retrieves user information from a music platform
func (h *Handler) handlePlatformUser(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")

	ctx, err := h.getRequestInjector(r)
	if err != nil {
		JSON(w, http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}

	provider, err := di.InvokeNamed[provider.Provider](ctx, platform)

	if err != nil {
		JSON(w, http.StatusBadRequest, &models.ErrorResponse{Error: err.Error()})
		return
	}
	user, err := provider.User(ctx)
	if err != nil {
		JSON(w, http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}
	JSON(w, http.StatusOK, user)
}
