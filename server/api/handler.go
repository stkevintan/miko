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
	ctx context.Context
}

func New(ctx context.Context) *Handler {
	return &Handler{ctx: ctx}
}

func (h *Handler) getApiRequestContext(r *http.Request) (context.Context, error) {
	ctx := r.Context()
	username := string(di.MustInvoke[models.Username](ctx))
	db := di.MustInvoke[*gorm.DB](ctx)
	cfg := di.MustInvoke[*config.Config](ctx)

	ctx = di.Inherit(ctx)

	var identity cookiecloud.Identity
	if err := db.Model(&cookiecloud.Identity{}).Select("username, uuid, password").Where("username = ?", username).First(&identity).Error; err != nil {
		return nil, fmt.Errorf("failed to find identity for user %s: %w", username, err)
	}

	jar, err := cookiecloud.NewCookieCloudJar(ctx, cfg.CookieCloud, &identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	di.Provide(ctx, jar)

	// Register providers in this scope so they can resolve the CookieJar
	neteaseProvider, err := netease.NewProvider(jar)
	if err != nil {
		return nil, fmt.Errorf("failed to create netease provider: %w", err)
	}

	di.ProvideNamed(ctx, "netease", neteaseProvider)

	return ctx, nil
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	// for docker health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/login", h.handleLogin)
		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(h.jwtAuth)
			r.Get("/me", h.handleGetMe)
			r.Post("/change-password", h.handleChangePassword)
			r.Get("/cookiecloud/server", h.getCookiecloudServer)
			r.Post("/cookiecloud/identity", h.handleCookiecloudIdentity)
			r.Post("/cookiecloud/pull", h.handleCookiecloudPull)
			r.Get("/download", h.handleDownload)
			r.Get("/platform/{platform}/user", h.handlePlatformUser)

			// Library
			r.Get("/library/folders", h.handleGetLibraryFolders)
			r.Get("/library/directory", h.handleGetLibraryDirectory)
			r.Get("/library/coverArt", h.handleGetLibraryCoverArt)
			r.Post("/library/scan", h.handleScanLibrary)
			r.Post("/library/scan/all", h.handleScanAllLibrary)
			r.Get("/library/scan/status", h.handleGetScanStatus)
			r.Get("/library/song/tags", h.handleGetLibrarySongTags)
			r.Post("/library/song/update", h.handleUpdateLibrarySong)
			r.Post("/library/song/cover", h.handleUpdateLibrarySongCover)
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
