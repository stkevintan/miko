package models

// ErrorResponse represents an error response
// @Description Error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid input"`
}

// LoginRequest represents the login request
// @Description User login request
type LoginRequest struct {
	Timeout  int    `json:"timeout,omitempty" example:"30000" description:"Timeout in milliseconds"`
	Server   string `json:"server,omitempty" example:"netease" description:"Music service server"`
	UUID     string `json:"uuid" binding:"required" example:"user123" description:"User identifier"`
	Password string `json:"password" binding:"required" example:"password123" description:"User password"`
}

// LoginResponse represents the login response
// @Description User login response
type LoginResponse struct {
	Username string `json:"username" example:"john_doe" description:"Username"`
	UserID   int64  `json:"user_id" example:"123456789" description:"User ID"`
	Success  bool   `json:"success" example:"true" description:"Login success status"`
	Message  string `json:"message,omitempty" example:"Login successful" description:"Additional message"`
}
