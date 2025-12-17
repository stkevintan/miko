package models

// HealthResponse represents the health check response
// @Description Health check response
type HealthResponse struct {
	Status      string `json:"status" example:"healthy"`
	Environment string `json:"environment" example:"development"`
}

// ProcessRequest represents the data processing request
// @Description Data processing request
type ProcessRequest struct {
	Data string `json:"data" binding:"required" example:"hello world"`
}

// ProcessResponse represents the data processing response
// @Description Data processing response
type ProcessResponse struct {
	Result string `json:"result" example:"processed: hello world"`
}

// ErrorResponse represents an error response
// @Description Error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid input"`
}
