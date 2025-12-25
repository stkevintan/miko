package models

// ErrorResponse represents an error response
// @Description Error response
type ErrorResponse struct {
	Error string `json:"error" example:"Invalid input"`
}

// @Description platform auth request
type CookiecloudIdentityRequest struct {
	Username string `json:"username" binding:"required" example:"your-username" description:"Associated username"`
	Key      string `gorm:"primaryKey" json:"key" binding:"required" example:"your-cookiecloud-key" description:"CookieCloud key"`
	Password string `json:"password" binding:"required" example:"your-cookiecloud-password" description:"CookieCloud password"`
}

// @Description platform auth response
type CookiecloudIdentityResponse struct {
	Key string `json:"key" binding:"required" example:"your-cookiecloud-key" description:"CookieCloud key"`
	Url string `json:"url" binding:"required" example:"https://cookiecloud.example.com" description:"CookieCloud server URL"`
}
