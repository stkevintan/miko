package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/server/api"
	"github.com/stkevintan/miko/server/subsonic"
)

// Handler contains HTTP handlers for our service
type Handler struct {
	ctx context.Context
}

// New creates a new handler instance
func New(ctx context.Context) *Handler {
	return &Handler{
		ctx: ctx,
	}
}

// Routes sets up the HTTP routes using Chi
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	cfg := di.MustInvoke[*config.Config](h.ctx)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Content-Length", "Accept", "Accept-Encoding", "X-CSRF-Token", "Authorization", "Range"},
		ExposedHeaders:   []string{"Accept-Ranges", "Content-Range", "Content-Length", "Content-Type", "ETag"},
		AllowCredentials: false,
	}))

	// Standard middleware
	if cfg.Log.Level == "debug" {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)

	// Subsonic .view suffix rewrite middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if strings.HasPrefix(path, "/rest") && strings.HasSuffix(path, ".view") {
				oldPath := path
				r.URL.Path = strings.TrimSuffix(path, ".view")
				log.Debug("Rewriting Subsonic path: %s -> %s", oldPath, r.URL.Path)
			}
			next.ServeHTTP(w, r)
		})
	})

	// subsonic v1.16.1 API group
	s := subsonic.New(h.ctx)
	s.RegisterRoutes(chi.Router(r))

	// API v1 group
	apiHandler := api.New(h.ctx)
	apiHandler.RegisterRoutes(chi.Router(r))

	return r
}
