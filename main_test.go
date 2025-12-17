package main

import (
	"testing"

	"github.com/stkevintan/miko/internal/config"
	"github.com/stkevintan/miko/internal/service"
)

func TestServiceIntegration(t *testing.T) {
	// Load test configuration
	cfg := &config.Config{
		Port:        8080,
		Environment: "test",
		LogLevel:    "debug",
	}

	// Initialize service
	svc := service.New(cfg)

	// Test health check
	health := svc.GetHealth()
	if health["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %s", health["status"])
	}

	if health["environment"] != "test" {
		t.Errorf("Expected environment 'test', got %s", health["environment"])
	}

	// Test data processing
	result, err := svc.ProcessData("test data")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expected := "processed: test data"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}
