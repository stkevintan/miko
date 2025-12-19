package service_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/internal/service"
)

func TestDownloadServiceIntegration(t *testing.T) {
	if os.Getenv("MIKO_INTEGRATION") == "" {
		t.Skip("Skipping integration test; set MIKO_INTEGRATION=1 to enable")
	}

	// Load config for testing
	cfg, err := config.Load()
	if err != nil {
		t.Skipf("Skipping integration test due to config load error: %v", err)
	}

	svc := service.New(cfg)

	t.Run("Download with URIs", func(t *testing.T) {
		outputDir := t.TempDir()
		tests := []struct {
			name      string
			options   *service.DownloadOptions
			wantError bool
			errorMsg  string
		}{
			{
				name: "valid song ID",
				options: &service.DownloadOptions{
					URIs:           []string{"123456"},
					Level:          "standard",
					Output:         outputDir,
					Timeout:        30 * time.Second,
					ConflictPolicy: "skip",
				},
				wantError: false,
			},
			{
				name: "multiple song IDs",
				options: &service.DownloadOptions{
					URIs:           []string{"123456", "789012"},
					Level:          "standard",
					Output:         outputDir,
					Timeout:        60 * time.Second,
					ConflictPolicy: "skip",
				},
				wantError: false,
			},
			{
				name: "song URL",
				options: &service.DownloadOptions{
					URIs:           []string{"https://music.163.com/song?id=123456"},
					Level:          "lossless",
					Output:         outputDir,
					Timeout:        30 * time.Second,
					ConflictPolicy: "skip",
				},
				wantError: false,
			},
			{
				name: "empty URIs",
				options: &service.DownloadOptions{
					URIs:           []string{},
					Level:          "standard",
					Output:         outputDir,
					Timeout:        30 * time.Second,
					ConflictPolicy: "skip",
				},
				wantError: true,
				errorMsg:  "URI",
			},
			{
				name: "invalid conflict policy",
				options: &service.DownloadOptions{
					URIs:           []string{"123456"},
					Level:          "standard",
					Output:         outputDir,
					Timeout:        30 * time.Second,
					ConflictPolicy: "invalid",
				},
				wantError: true,
				errorMsg:  "conflict policy",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := svc.Download(context.Background(), tt.options)

				if tt.wantError {
					if err == nil {
						t.Errorf("Expected error for %s", tt.name)
					} else if tt.errorMsg != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
						t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
					}
					return
				}

				// For real integration runs, the API may still fail (auth, geo, etc.).
				// If we got a result, validate internal consistency.
				if result != nil {
					total := result.Total()
					success := result.SuccessCount()
					failed := result.FailedCount()
					if success+failed != total {
						t.Errorf("Success (%d) + Failed (%d) != Total (%d)", success, failed, total)
					}
					if failed > 0 {
						foundErr := false
						for _, r := range result.Results {
							if r != nil && r.Err != nil {
								foundErr = true
								break
							}
						}
						if !foundErr {
							t.Error("Expected at least one per-item error when failed count > 0")
						}
					}
				}

				// If there's an error, log it but don't fail the integration test.
				if err != nil {
					t.Logf("Download attempt returned error (may be expected in integration env): %v", err)
				}
			})
		}
	})

	t.Run("Download with different quality levels", func(t *testing.T) {
		qualityLevels := []string{"standard", "higher", "exhigh", "lossless", "hires"}
		outputDir := t.TempDir()

		for _, level := range qualityLevels {
			t.Run(level, func(t *testing.T) {
				options := &service.DownloadOptions{
					URIs:           []string{"123456"},
					Level:          level,
					Output:         outputDir,
					Timeout:        30 * time.Second,
					ConflictPolicy: "skip",
				}

				result, err := svc.Download(context.Background(), options)

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
					level, result.Total(), result.SuccessCount(), result.FailedCount())
			})
		}
	})

	t.Run("Download with different conflict policies", func(t *testing.T) {
		policies := []string{"skip", "overwrite", "rename", "update_tags"}
		outputDir := t.TempDir()

		for _, policy := range policies {
			t.Run(policy, func(t *testing.T) {
				options := &service.DownloadOptions{
					URIs:           []string{"123456"},
					Level:          "standard",
					Output:         outputDir,
					Timeout:        30 * time.Second,
					ConflictPolicy: policy,
				}

				result, err := svc.Download(context.Background(), options)

				if err != nil {
					t.Logf("Download with policy %s failed: %v", policy, err)
					return
				}

				if result == nil {
					t.Errorf("Download result is nil for policy %s", policy)
					return
				}

				t.Logf("Download with policy %s: Total=%d, Success=%d, Failed=%d",
					policy, result.Total(), result.SuccessCount(), result.FailedCount())
			})
		}
	})

	t.Run("Download with timeout", func(t *testing.T) {
		outputDir := t.TempDir()
		// Test very short timeout
		options := &service.DownloadOptions{
			URIs:           []string{"123456"},
			Level:          "lossless",
			Output:         outputDir,
			Timeout:        1 * time.Millisecond, // Very short timeout
			ConflictPolicy: "skip",
		}

		_, err := svc.Download(context.Background(), options)

		// We expect this to likely timeout, but it's not guaranteed
		if err != nil {
			t.Logf("Download with short timeout failed as expected: %v", err)
		} else {
			t.Log("Download with short timeout unexpectedly succeeded")
		}
	})
}
