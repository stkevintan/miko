package service

import (
	"github.com/stkevintan/miko/internal/config"
)

// Service contains the business logic for our application
type Service struct {
	config *config.Config
}

// New creates a new service instance
func New(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
	}
}

// GetHealth returns the health status of the service
func (s *Service) GetHealth() map[string]string {
	return map[string]string{
		"status":      "healthy",
		"environment": s.config.Environment,
	}
}

// Example business logic method
func (s *Service) ProcessData(data string) (string, error) {
	// Add your business logic here
	return "processed: " + data, nil
}
