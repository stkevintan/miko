package downloader

import (
	"context"
	"strings"
	"testing"

	"github.com/stkevintan/miko/internal/config"
	"github.com/stkevintan/miko/internal/types"
)

func TestDownloaderManager(t *testing.T) {
	t.Run("NewDownloaderManager", func(t *testing.T) {
		manager := NewDownloaderManager()

		if manager == nil {
			t.Fatal("NewDownloaderManager returned nil")
		}

		if manager.factories == nil {
			t.Error("factories map not initialized")
		}

		// Test default factories are registered
		platforms := manager.GetSupportedPlatforms()
		if len(platforms) == 0 {
			t.Error("No default factories registered")
		}

		// Should have netease and 163
		expectedPlatforms := map[string]bool{
			"netease": false,
			"163":     false,
		}

		for _, platform := range platforms {
			if _, exists := expectedPlatforms[platform]; exists {
				expectedPlatforms[platform] = true
			}
		}

		for platform, found := range expectedPlatforms {
			if !found {
				t.Errorf("Expected platform %q not found", platform)
			}
		}
	})

	t.Run("RegisterFactory", func(t *testing.T) {
		manager := NewDownloaderManager()

		// Create a mock factory
		mockFactory := &NetEaseDownloaderFactory{}

		// Register new platform
		manager.RegisterFactory("test", mockFactory)

		platforms := manager.GetSupportedPlatforms()
		found := false
		for _, platform := range platforms {
			if platform == "test" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Registered platform 'test' not found in supported platforms")
		}
	})

	t.Run("CreateDownloader", func(t *testing.T) {
		manager := NewDownloaderManager()

		// Try to load config for testing
		cfg, err := config.Load()
		if err != nil {
			t.Skipf("Skipping CreateDownloader test due to config load error: %v", err)
		}

		tests := []struct {
			name      string
			platform  string
			config    *types.DownloaderConfig
			wantError bool
			errorMsg  string
		}{
			{
				name:     "valid netease platform",
				platform: "netease",
				config: &types.DownloaderConfig{
					Level:          "standard",
					Output:         "./test",
					ConflictPolicy: "skip",
					Root:           cfg,
				},
				wantError: false,
			},
			{
				name:     "valid 163 platform",
				platform: "163",
				config: &types.DownloaderConfig{
					Level:          "standard",
					Output:         "./test",
					ConflictPolicy: "skip",
					Root:           cfg,
				},
				wantError: false,
			},
			{
				name:      "unsupported platform",
				platform:  "spotify",
				config:    &types.DownloaderConfig{},
				wantError: true,
				errorMsg:  "unsupported platform",
			},
			{
				name:     "invalid config",
				platform: "netease",
				config: &types.DownloaderConfig{
					Level:          "invalid_level",
					Output:         "./test",
					ConflictPolicy: "skip",
					Root:           cfg,
				},
				wantError: true,
				errorMsg:  "invalid quality level",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				downloader, err := manager.CreateDownloader(context.Background(), tt.platform, tt.config)

				if tt.wantError {
					if err == nil {
						t.Errorf("Expected error for %s", tt.name)
					} else if tt.errorMsg != "" && !containsIgnoreCaseManager(err.Error(), tt.errorMsg) {
						t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
					}
					return
				}

				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
					return
				}

				if downloader == nil {
					t.Errorf("CreateDownloader returned nil downloader for %s", tt.name)
				}
			})
		}
	})

	t.Run("GetSupportedPlatforms", func(t *testing.T) {
		manager := NewDownloaderManager()

		platforms := manager.GetSupportedPlatforms()
		if len(platforms) == 0 {
			t.Error("GetSupportedPlatforms returned empty slice")
		}

		// Test that platforms are unique
		seen := make(map[string]bool)
		for _, platform := range platforms {
			if seen[platform] {
				t.Errorf("Duplicate platform found: %s", platform)
			}
			seen[platform] = true
		}

		// Test that all registered platforms are returned
		manager.RegisterFactory("custom", &NetEaseDownloaderFactory{})
		newPlatforms := manager.GetSupportedPlatforms()

		if len(newPlatforms) != len(platforms)+1 {
			t.Errorf("Expected %d platforms, got %d", len(platforms)+1, len(newPlatforms))
		}

		found := false
		for _, platform := range newPlatforms {
			if platform == "custom" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Custom platform not found after registration")
		}
	})
}

// Helper function to check if string contains substring (case insensitive)
func containsIgnoreCaseManager(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
