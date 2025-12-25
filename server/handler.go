package server

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/server/api"
	"github.com/stkevintan/miko/server/subsonic"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handler contains HTTP handlers for our service
type Handler struct {
	injector do.Injector
}

// New creates a new handler instance
func New(i do.Injector) *Handler {
	return &Handler{
		injector: i,
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

	// subsonic v1.16.1 API group
	s := subsonic.New(h.injector)
	s.RegisterRoutes(r)

	// API v1 group
	apiHandler := api.New(h.injector)
	apiHandler.RegisterRoutes(r)

	return r
}
