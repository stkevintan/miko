package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/pkg/registry"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handler contains HTTP handlers for our service
type Handler struct {
	registry *registry.ProviderRegistry
}

// New creates a new handler instance
func New(registry *registry.ProviderRegistry) *Handler {
	return &Handler{
		registry: registry,
	}
}

// Routes sets up the HTTP routes using Gin
func (h *Handler) Routes() *gin.Engine {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Swagger UI endpoint
	r.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	// Redirect /docs to /docs/index.html for convenience
	r.GET("/docs", func(c *gin.Context) {
		c.Redirect(301, "/docs/index.html")
	})

	// API group
	api := r.Group("/api")
	{
		api.POST("/login", h.handleLogin)
		api.GET("/download", h.handleDownload)
	}

	return r
}
