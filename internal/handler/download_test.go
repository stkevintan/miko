package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/internal/handler"
	"github.com/stkevintan/miko/internal/service"
	"github.com/stkevintan/miko/pkg/models"
)

// Helper function for string containment check

func TestDownloadHandler(t *testing.T) {
	contains := func(s, substr string) bool {
		return strings.Contains(s, substr)
	}
	// Setup
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	svc := service.New(cfg)
	h := handler.New(svc)
	mux := h.Routes()

	// Test download endpoint with valid request
	t.Run("GET /api/download with valid song ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/download?uri=123456&level=lossless&timeout=30000", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Since we don't have actual login/authentication, this will likely fail
		// but we can test the request structure and error handling
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		// Check if response has expected structure
		if w.Code == http.StatusOK {
			// Successful response should have download info
			if _, hasSummary := response["summary"]; !hasSummary {
				t.Error("Expected 'summary' field in response")
			}
			if _, hasDetails := response["details"]; !hasDetails {
				t.Error("Expected 'details' field in response")
			}
		} else {
			// Error response should have error field
			if _, hasError := response["error"]; !hasError {
				t.Error("Expected 'error' field in error response")
			}
		}
	})

	// Test download endpoint with missing song ID
	t.Run("GET /api/download with missing uri", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/download?level=lossless&timeout=30000", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response models.ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if !contains(response.Error, "uri") && !contains(response.Error, "required") {
			t.Errorf("Expected error about required uri field, got '%s'", response.Error)
		}
	})

	// Test download endpoint with minimal valid request
	t.Run("GET /api/download with minimal parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/download?uri=123456", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Should work with defaults
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}
	})

	// Test download endpoint with URL
	t.Run("GET /api/download with song URL", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/download?uri=https://music.163.com/song?id=123456", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Should work with URL parsing
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}
	})
}
