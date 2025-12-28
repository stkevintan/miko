package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/provider"
)

// handleUser retrieves user information from a music platform
func (h *Handler) handlePlatformUser(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")

	injector, err := h.getRequestInjector(r)
	if err != nil {
		JSON(w, http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}
	defer injector.Shutdown()

	provider, err := do.InvokeNamed[provider.Provider](injector, platform)

	if err != nil {
		JSON(w, http.StatusBadRequest, &models.ErrorResponse{Error: err.Error()})
		return
	}
	user, err := provider.User(r.Context())
	if err != nil {
		JSON(w, http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}
	JSON(w, http.StatusOK, user)
}
