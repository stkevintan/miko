package models

// ErrorResponse represents an error response
// @Description Error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid input"`
}

// @Description User login response
type LoginResponse struct {
	Username string `json:"username" example:"john_doe" description:"Username"`
	UserID   int64  `json:"user_id" example:"123456789" description:"User ID"`
	Success  bool   `json:"success" example:"true" description:"Login success status"`
	Message  string `json:"message,omitempty" example:"Login successful" description:"Additional message"`
}
