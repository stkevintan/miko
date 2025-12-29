package api

import (
	"encoding/json"
	"net/http"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/di"
	"gorm.io/gorm"
)

// handleCookiecloudIdentity updates CookieCloud identity
func (h *Handler) handleCookiecloudIdentity(w http.ResponseWriter, r *http.Request) {

	var req models.CookiecloudIdentityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		JSON(w, http.StatusBadRequest, errorResp)
		return
	}

	username := string(di.MustInvoke[models.Username](r.Context()))
	db := di.MustInvoke[*gorm.DB](r.Context())

	// Save the identity to database associated with current account
	identity := cookiecloud.Identity{
		Username: username,
		UUID:     req.Key,
		Password: req.Password,
	}

	if err := db.Save(&identity).Error; err != nil {
		errorResp := models.ErrorResponse{Error: "Failed to save identity: " + err.Error()}
		JSON(w, http.StatusInternalServerError, errorResp)
		return
	}

	// Force pull after identity saved
	ctx, err := h.getApiRequestContext(r)
	if err == nil {
		if jar, err := di.Invoke[cookiecloud.CookieJar](ctx); err == nil {
			_ = jar.PullAll()
		}
	}

	cfg := di.MustInvoke[*config.Config](r.Context())

	response := models.CookiecloudIdentityResponse{
		Url: cfg.CookieCloud.Url,
		Key: req.Key,
	}

	JSON(w, http.StatusOK, response)
}

// handleCookiecloudPull forces a pull from CookieCloud
func (h *Handler) handleCookiecloudPull(w http.ResponseWriter, r *http.Request) {
	ctx, err := h.getApiRequestContext(r)
	if err != nil {
		JSON(w, http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}

	jar, err := di.Invoke[cookiecloud.CookieJar](ctx)
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
	cfg := di.MustInvoke[*config.Config](r.Context())
	JSON(w, http.StatusOK, map[string]string{"url": cfg.CookieCloud.Url})
}
