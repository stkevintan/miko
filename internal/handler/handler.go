package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/internal/models"
	"github.com/stkevintan/miko/internal/service"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handler contains HTTP handlers for our service
type Handler struct {
	service *service.Service
}

// New creates a new handler instance
func New(svc *service.Service) *Handler {
	return &Handler{
		service: svc,
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
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API group
	api := r.Group("/api")
	{
		// Health endpoint
		api.GET("/health", h.handleHealth)

		// Business logic endpoints
		api.POST("/process", h.handleProcess)
		api.POST("/login", h.handleLogin)
		api.GET("/download", h.handleDownload)
	}

	return r
}

// handleHealth returns the health status
// @Summary      Get health status
// @Description  Returns the current health status of the service
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200  {object}  models.HealthResponse
// @Router       /health [get]
func (h *Handler) handleHealth(c *gin.Context) {
	health := h.service.GetHealth()
	response := models.HealthResponse{
		Status:      health["status"],
		Environment: health["environment"],
	}

	c.JSON(http.StatusOK, response)
}

// handleProcess processes data
// @Summary      Process data
// @Description  Processes input data and returns the result
// @Tags         data
// @Accept       json
// @Produce      json
// @Param        request body models.ProcessRequest true "Data to process"
// @Success      200 {object} models.ProcessResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /process [post]
func (h *Handler) handleProcess(c *gin.Context) {
	var req models.ProcessRequest

	// ShouldBindJSON automatically handles JSON parsing and validation
	if err := c.ShouldBindJSON(&req); err != nil {
		// Check if it's a validation error for required field
		if req.Data == "" {
			errorResp := models.ErrorResponse{Error: "Data field is required"}
			c.JSON(http.StatusBadRequest, errorResp)
			return
		}
		errorResp := models.ErrorResponse{Error: "Invalid JSON: " + err.Error()}
		c.JSON(http.StatusBadRequest, errorResp)
		return
	}

	// Process the data
	result, err := h.service.ProcessData(req.Data)
	if err != nil {
		errorResp := models.ErrorResponse{Error: "Processing failed: " + err.Error()}
		c.JSON(http.StatusInternalServerError, errorResp)
		return
	}

	// Return success response
	response := models.ProcessResponse{
		Result: result,
	}
	c.JSON(http.StatusOK, response)
}
