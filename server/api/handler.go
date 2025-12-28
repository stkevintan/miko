package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/netease"
	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	cfg       *config.Config
	injector  do.Injector
	jwtSecret []byte
}

func New(i do.Injector) *Handler {
	return &Handler{
		db:       do.MustInvoke[*gorm.DB](i),
		cfg:      do.MustInvoke[*config.Config](i),
		injector: i,
	}
}

func (h *Handler) getRequestInjector(r *http.Request) (do.Injector, error) {
	username := models.GetUsername(r)

	var identity cookiecloud.Identity
	if err := h.db.Where("username = ?", username).First(&identity).Error; err != nil {
		return nil, fmt.Errorf("failed to find identity for user %s: %w", username, err)
	}

	jar, err := cookiecloud.NewCookieCloudJar(r.Context(), h.cfg.CookieCloud, &identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Create a request-scoped injector
	scope := h.injector.Scope(fmt.Sprintf("request-%s-%d", username, time.Now().UnixNano()))
	do.Provide(scope, func(i do.Injector) (cookiecloud.CookieJar, error) {
		return jar, nil
	})

	// Register providers in this scope so they can resolve the CookieJar
	do.ProvideNamed(scope, "netease", netease.NewNetEaseProvider)

	return scope, nil
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/api", func(r chi.Router) {
		r.Post("/login", h.handleLogin)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(h.jwtAuth)
			r.Get("/cookiecloud/server", h.getCookiecloudServer)
			r.Post("/cookiecloud/identity", h.handleCookiecloudIdentity)
			r.Post("/cookiecloud/pull", h.handleCookiecloudPull)
			r.Get("/download", h.handleDownload)
			r.Get("/platform/{platform}/user", h.handlePlatformUser)
		})
	})
}

// JSON writes a JSON response with the given status code and data
func JSON(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

// Error writes a plain text error response
func Error(w http.ResponseWriter, code int, message string) {
	http.Error(w, message, code)
}
