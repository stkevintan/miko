package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/netease"
	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	cfg       *config.Config
	ctx       context.Context
	jwtSecret []byte
}

func New(ctx context.Context) *Handler {
	return &Handler{
		db:  di.MustInvoke[*gorm.DB](ctx),
		cfg: di.MustInvoke[*config.Config](ctx),
		ctx: ctx,
	}
}

func (h *Handler) getRequestInjector(r *http.Request) (context.Context, error) {
	username, err := models.GetUsername(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get username from request: %w", err)
	}

	var identity cookiecloud.Identity
	if err := h.db.Where("username = ?", username).First(&identity).Error; err != nil {
		return nil, fmt.Errorf("failed to find identity for user %s: %w", username, err)
	}

	jar, err := cookiecloud.NewCookieCloudJar(r.Context(), h.cfg.CookieCloud, &identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	// Create a request-scoped context that inherits global dependencies
	ctx := di.NewScope(h.ctx)
	di.Provide(ctx, jar)

	// Register providers in this scope so they can resolve the CookieJar
	neteaseProvider, err := netease.NewNetEaseProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create netease provider: %w", err)
	}
	di.ProvideNamed(ctx, "netease", neteaseProvider)

	return ctx, nil
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
