package service

import (
	"context"
	"strings"
	"testing"
	"time"

	l "github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/stkevintan/miko/internal/config"
)

func TestDownloadServiceIntegration(t *testing.T) {
	// Initialize logger to prevent nil pointer panics
	l.Default = l.New(&l.Config{
		Level:  "info",
		Format: "text",
		Stdout: true,
	})

	// Load config for testing
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping integration test due to config load error: %v", err)
	}

	service := New(cfg)

	t.Run("Download with URIs", func(t *testing.T) {
		tests := []struct {
			name      string
			options   *DownloadOptions
			wantError bool
			errorMsg  string
		}{
			{
				name: "valid song ID",
				options: &DownloadOptions{
					URIs:           []string{"123456"},
					Level:          "standard",
					Output:         "./test_downloads",
					Timeout:        30 * time.Second,
					ConflictPolicy: "skip",
				},
				wantError: false,
			},
			{
				name: "multiple song IDs",
				options: &DownloadOptions{
					URIs:           []string{"123456", "789012"},
					Level:          "standard",
					Output:         "./test_downloads",
					Timeout:        60 * time.Second,
					ConflictPolicy: "skip",
				},
				wantError: false,
			},
			{
				name: "song URL",
				options: &DownloadOptions{
					URIs:           []string{"https://music.163.com/song?id=123456"},
					Level:          "lossless",
					Output:         "./test_downloads",
					Timeout:        30 * time.Second,
					ConflictPolicy: "skip",
				},
				wantError: false,
			},
			{
				name: "empty URIs",
				options: &DownloadOptions{
					URIs:           []string{},
					Level:          "standard",
					Output:         "./test_downloads",
					Timeout:        30 * time.Second,
					ConflictPolicy: "skip",
				},
				wantError: true,
				errorMsg:  "URI",
			},
			{
				name: "invalid conflict policy",
				options: &DownloadOptions{
					URIs:           []string{"123456"},
					Level:          "standard",
					Output:         "./test_downloads",
					Timeout:        30 * time.Second,
					ConflictPolicy: "invalid",
				},
				wantError: true,
				errorMsg:  "conflict policy",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := service.Download(context.Background(), tt.options)

				if tt.wantError {
					if err == nil {
						t.Errorf("Expected error for %s", tt.name)
					} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
						t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
					}
					return
				}

				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
					return
				}

				if result == nil {
					t.Errorf("Download result is nil for %s", tt.name)
					return
				}

				// Validate result structure
				expectedTotal := int64(len(tt.options.URIs))
				if result.Total != expectedTotal {
					t.Errorf("Expected total %d, got %d", expectedTotal, result.Total)
				}

				if result.Success+result.Failed != result.Total {
					t.Errorf("Success (%d) + Failed (%d) != Total (%d)",
						result.Success, result.Failed, result.Total)
				}

				if len(result.Songs) != int(result.Success) {
					t.Errorf("Expected %d songs in result, got %d",
						result.Success, len(result.Songs))
				}

				if result.Failed > 0 && len(result.Errors) == 0 {
					t.Error("Expected error messages when failed count > 0")
				}
			})
		}
	})

	t.Run("Download with different quality levels", func(t *testing.T) {
		qualityLevels := []string{"standard", "higher", "exhigh", "lossless", "hires"}

		for _, level := range qualityLevels {
			t.Run(level, func(t *testing.T) {
				options := &DownloadOptions{
					URIs:           []string{"123456"},
					Level:          level,
					Output:         "./test_downloads",
					Timeout:        30 * time.Second,
					ConflictPolicy: "skip",
				}

				result, err := service.Download(context.Background(), options)

				if err != nil {
					t.Logf("Download with level %s failed: %v", level, err)
					// Don't fail the test as some quality levels might not be available
					return
				}

				if result == nil {
					t.Errorf("Download result is nil for level %s", level)
					return
				}

				t.Logf("Download with level %s: Total=%d, Success=%d, Failed=%d",
					level, result.Total, result.Success, result.Failed)
			})
		}
	})

	t.Run("Download with different conflict policies", func(t *testing.T) {
		policies := []string{"skip", "overwrite", "rename", "update_tags"}

		for _, policy := range policies {
			t.Run(policy, func(t *testing.T) {
				options := &DownloadOptions{
					URIs:           []string{"123456"},
					Level:          "standard",
					Output:         "./test_downloads",
					Timeout:        30 * time.Second,
					ConflictPolicy: policy,
				}

				result, err := service.Download(context.Background(), options)

				if err != nil {
					t.Logf("Download with policy %s failed: %v", policy, err)
					return
				}

				if result == nil {
					t.Errorf("Download result is nil for policy %s", policy)
					return
				}

				t.Logf("Download with policy %s: Total=%d, Success=%d, Failed=%d",
					policy, result.Total, result.Success, result.Failed)
			})
		}
	})

	t.Run("Download with timeout", func(t *testing.T) {
		// Test very short timeout
		options := &DownloadOptions{
			URIs:           []string{"123456"},
			Level:          "lossless",
			Output:         "./test_downloads",
			Timeout:        1 * time.Millisecond, // Very short timeout
			ConflictPolicy: "skip",
		}

		_, err := service.Download(context.Background(), options)

		// We expect this to likely timeout, but it's not guaranteed
		if err != nil {
			t.Logf("Download with short timeout failed as expected: %v", err)
		} else {
			t.Log("Download with short timeout unexpectedly succeeded")
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && containsIgnoreCase(s, substr)
}

func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains check
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if strings.ToLower(string(s[i+j])) != strings.ToLower(string(substr[j])) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
