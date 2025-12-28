package api

import (
	"encoding/json"
	"net/http"

	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/cookiecloud"
)

// handleCookiecloudIdentity updates CookieCloud identity
func (h *Handler) handleCookiecloudIdentity(w http.ResponseWriter, r *http.Request) {

	var req models.CookiecloudIdentityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		JSON(w, http.StatusBadRequest, errorResp)
		return
	}

	username := models.GetUsername(r)

	// Save the identity to database associated with current account
	identity := cookiecloud.Identity{
		Username: username,
		UUID:     req.Key,
		Password: req.Password,
	}

	if err := h.db.Save(&identity).Error; err != nil {
		errorResp := models.ErrorResponse{Error: "Failed to save identity: " + err.Error()}
		JSON(w, http.StatusInternalServerError, errorResp)
		return
	}

	// Force pull after identity saved
	injector, err := h.getRequestInjector(r)
	if err == nil {
		if jar, err := do.Invoke[cookiecloud.CookieJar](injector); err == nil {
			_ = jar.PullAll()
		}
	}

	response := models.CookiecloudIdentityResponse{
		Url: h.cfg.CookieCloud.Url,
		Key: req.Key,
	}

	JSON(w, http.StatusOK, response)
}

// handleCookiecloudPull forces a pull from CookieCloud
func (h *Handler) handleCookiecloudPull(w http.ResponseWriter, r *http.Request) {
	injector, err := h.getRequestInjector(r)
	if err != nil {
		JSON(w, http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}

	jar, err := do.Invoke[cookiecloud.CookieJar](injector)
	if err != nil {
		JSON(w, http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}

	if err := jar.PullAll(); err != nil {
		JSON(w, http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "Cookies pulled successfully"})
}

// getCookiecloudServer retrieves the CookieCloud server URL
func (h *Handler) getCookiecloudServer(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, map[string]string{"url": h.cfg.CookieCloud.Url})
}
