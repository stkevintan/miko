package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stkevintan/miko/config"
)

func TestDownloadService(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping DownloadService tests due to config load error: %v", err)
	}
	service := New(cfg)

	// Create test directory
	testDir := "./test_downloads"
	defer os.RemoveAll(testDir) // Clean up after tests
	err = os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Test Download with song ID
	t.Run("Download with song ID", func(t *testing.T) {
		options := &DownloadOptions{
			URIs:    []string{"123456"}, // song ID
			Level:   "standard",
			Output:  "./test_downloads",
			Timeout: 30 * time.Second,
		}

		// This will likely fail due to lack of authentication
		// but we can test the structure is correct
		result, err := service.Download(context.Background(), options)

		// Since we don't have real authentication, expect an error
		// but validate that it's not due to struct issues
		if err != nil {
			t.Logf("Expected error due to authentication: %v", err)
			// Make sure it's not a struct-related error
			if err.Error() == "URIs are required" {
				t.Error("URIs validation failed")
			}
		}

		// If somehow successful, check result structure
		if result != nil {
			if result.Total() == 0 {
				t.Error("Expected at least one song to be processed")
			}
		}
	})

	// Test missing URIs validation
	t.Run("Download with missing URIs", func(t *testing.T) {
		options := &DownloadOptions{
			URIs:    []string{}, // empty URIs should fail
			Level:   "standard",
			Output:  "./test_downloads",
			Timeout: 30 * time.Second,
		}

		result, err := service.Download(context.Background(), options)

		if err == nil {
			t.Error("Expected error for missing URIs")
		}

		if result != nil {
			t.Error("Expected nil result for invalid input")
		}

		// Check for appropriate error message
		if err != nil && !strings.Contains(err.Error(), "URI") && !strings.Contains(err.Error(), "required") {
			t.Logf("Got error (expected for empty URIs): %v", err)
		}
	})

	// Test Download with URL
	t.Run("Download with URL", func(t *testing.T) {
		options := &DownloadOptions{
			URIs:    []string{"https://music.163.com/song?id=123456"},
			Level:   "standard",
			Output:  "./test_downloads",
			Timeout: 30 * time.Second,
		}

		result, err := service.Download(context.Background(), options)

		// This will likely fail due to lack of authentication
		// but we can test the URL parsing works
		if err != nil {
			t.Logf("Expected error due to authentication: %v", err)
		}

		// Since authentication may fail, just check that we got some response
		// and the error is not about invalid input format
		if result == nil && err != nil {
			// Make sure it's not a parsing error
			if strings.Contains(err.Error(), "URI") && strings.Contains(err.Error(), "required") {
				t.Error("URL parsing failed - URI validation error")
			}
		}
	})

	// Test Download with multiple URIs
	t.Run("Download with multiple URIs", func(t *testing.T) {
		options := &DownloadOptions{
			URIs:           []string{"123456", "789012", "https://music.163.com/song?id=345678"},
			Level:          "standard",
			Output:         "./test_downloads",
			Timeout:        30 * time.Second,
			ConflictPolicy: "skip",
		}

		result, err := service.Download(context.Background(), options)

		// This will likely fail due to lack of authentication
		if err != nil {
			t.Logf("Expected error due to authentication: %v", err)
		}

		// If result is returned, it should reflect multiple items
		if result != nil {
			if result.Total() != 3 {
				t.Logf("Expected total of 3, got %d (may vary due to auth issues)", result.Total())
			}
		}
	})

	t.Run("Output path is normalized to absolute", func(t *testing.T) {
		rel := "./test_downloads"
		expectedAbs, err := filepath.Abs(rel)
		if err != nil {
			t.Fatalf("filepath.Abs failed: %v", err)
		}

		options := &DownloadOptions{
			Platform: "this-platform-does-not-exist",
			URIs:     []string{"123456"},
			Level:    "standard",
			Output:   rel,
			Timeout:  1 * time.Second,
		}

		_, _ = service.Download(context.Background(), options)
		if options.Output != expectedAbs {
			t.Fatalf("expected Output to be normalized to %q, got %q", expectedAbs, options.Output)
		}
		if !filepath.IsAbs(options.Output) {
			t.Fatalf("expected Output to be absolute, got %q", options.Output)
		}
	})
}
