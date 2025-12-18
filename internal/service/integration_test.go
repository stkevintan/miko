package service

import (
	"testing"

	"github.com/stkevintan/miko/internal/config"
)

func TestServiceIntegration(t *testing.T) {
	// Test basic service creation and health check
	t.Run("Service creation and health", func(t *testing.T) {
		cfg := &config.Config{
			Environment: "test",
		}
		service := New(cfg)

		if service == nil {
			t.Error("Expected service to be created")
		}

		health := service.GetHealth()
		if health["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", health["status"])
		}

		if health["environment"] != "test" {
			t.Errorf("Expected environment 'test', got '%s'", health["environment"])
		}
	})
}
