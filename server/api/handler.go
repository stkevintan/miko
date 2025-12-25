package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/pkg/cookiecloud"
	"gorm.io/gorm"
)

type Handler struct {
	db       *gorm.DB
	cfg      *config.Config
	injector do.Injector
}

func New(i do.Injector) *Handler {
	return &Handler{
		db:       do.MustInvoke[*gorm.DB](i),
		cfg:      do.MustInvoke[*config.Config](i),
		injector: i,
	}
}

func (h *Handler) getRequestInjector(c *gin.Context) (do.Injector, error) {
	username, ok := c.Get("username")
	if !ok {
		return nil, fmt.Errorf("username not found in context")
	}

	var identity cookiecloud.Identity
	if err := h.db.Where("username = ?", username).First(&identity).Error; err != nil {
		return nil, fmt.Errorf("failed to find identity for user %s: %w", username, err)
	}

	jar, err := cookiecloud.NewCookieCloudJar(c.Request.Context(), h.cfg.CookieCloud, h.db, &identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create cookie jar: %w", err)
	}

	scope := h.injector.Scope(username.(string))
	do.Provide(scope, func(i do.Injector) (cookiecloud.CookieJar, error) {
		return jar, nil
	})
	return scope, nil
}
func (h *Handler) RegisterRoutes(r *gin.Engine) *gin.RouterGroup {
	// api
	api := r.Group("/api")
	{
		api.POST("/login", h.handleLogin)

		// Protected routes
		protected := api.Group("/")
		protected.Use(h.authMiddleware())
		{
			protected.GET("/cookiecloud/server", h.getCookiecloudServer)
			protected.POST("/cookiecloud/identity", h.handleCookiecloudIdentity)
			protected.GET("/download", h.handleDownload)
			protected.GET("/platform/:platform/user", h.handlePlatformUser)
		}
	}
	return api
}
