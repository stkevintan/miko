package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stkevintan/miko/internal/config"
	"github.com/stkevintan/miko/internal/models"
	"github.com/stkevintan/miko/internal/service"
)

// Helper function for string containment check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestHandler(t *testing.T) {
	// Setup
	cfg := &config.Config{
		Port:        8080,
		Environment: "test",
		LogLevel:    "debug",
	}
	svc := service.New(cfg)
	handler := New(svc)
	mux := handler.Routes()

	// Test health endpoint
	t.Run("GET /api/health", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response models.HealthResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response.Status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%s'", response.Status)
		}

		if response.Environment != "test" {
			t.Errorf("Expected environment 'test', got '%s'", response.Environment)
		}
	})

	// Test process endpoint
	t.Run("POST /api/process", func(t *testing.T) {
		requestBody := models.ProcessRequest{
			Data: "test data",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/api/process", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response models.ProcessResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		expected := "processed: test data"
		if response.Result != expected {
			t.Errorf("Expected result '%s', got '%s'", expected, response.Result)
		}
	})

	// Test process endpoint with empty data
	t.Run("POST /api/process with empty data", func(t *testing.T) {
		requestBody := models.ProcessRequest{
			Data: "",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req := httptest.NewRequest("POST", "/api/process", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response models.ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if response.Error != "Data field is required" {
			// Also accept Gin's validation message
			if !contains(response.Error, "Data") && !contains(response.Error, "required") {
				t.Errorf("Expected error about required Data field, got '%s'", response.Error)
			}
		}
	})
}
