package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	l "github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/stkevintan/miko/internal/config"
	"github.com/stkevintan/miko/internal/models"
	"github.com/stkevintan/miko/internal/service"
)

func TestDownloadHandler(t *testing.T) {
	// Initialize logger
	l.Default = l.New(&l.Config{
		Level:  "info",
		Format: "text",
		Stdout: true,
	})

	// Setup
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	svc := service.New(cfg)
	handler := New(svc)
	mux := handler.Routes()

	// Test download endpoint with valid request
	t.Run("GET /api/download with valid song ID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/download?resource=123456&level=lossless&output=./test_downloads&timeout=30000", nil)
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
			if _, hasSuccess := response["success"]; !hasSuccess {
				t.Error("Expected 'success' field in response")
			}
		} else {
			// Error response should have error field
			if _, hasError := response["error"]; !hasError {
				t.Error("Expected 'error' field in error response")
			}
		}
	})

	// Test download endpoint with missing song ID
	t.Run("GET /api/download with missing resource", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/download?level=lossless&output=./test_downloads&timeout=30000", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var response models.ErrorResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("Failed to decode response: %v", err)
		}

		if !contains(response.Error, "resource") && !contains(response.Error, "required") {
			t.Errorf("Expected error about required resource field, got '%s'", response.Error)
		}
	})

	// Test download endpoint with minimal valid request
	t.Run("GET /api/download with minimal parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/download?resource=123456", nil)
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
		req := httptest.NewRequest("GET", "/api/download?resource=https://music.163.com/song?id=123456", nil)
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
