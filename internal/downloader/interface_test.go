package downloader_test

import (
	"context"
	"testing"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/internal/downloader"
	"github.com/stkevintan/miko/pkg/types"
)

func TestDownloaderInterface(t *testing.T) {
	// Test the downloader manager
	t.Run("DownloaderManager", func(t *testing.T) {
		manager := downloader.NewDownloaderManager()

		// Test supported platforms
		platforms := manager.GetSupportedPlatforms()
		if len(platforms) == 0 {
			t.Error("Expected at least one supported platform")
		}

		expectedPlatforms := []string{"netease", "163"}
		for _, expected := range expectedPlatforms {
			found := false
			for _, platform := range platforms {
				if platform == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected platform %q to be supported", expected)
			}
		}

		// Test creating downloader for unsupported platform
		_, err := manager.CreateDownloader(context.Background(), "unsupported", &types.DownloaderConfig{})
		if err == nil {
			t.Error("Expected error for unsupported platform")
		}
	})

	// Test factory implementation
	t.Run("NetEaseDownloaderFactory", func(t *testing.T) {
		factory := &downloader.NetEaseDownloaderFactory{}

		// Test supported platforms
		platforms := factory.SupportedPlatforms()
		expectedPlatforms := []string{"netease", "163"}

		if len(platforms) != len(expectedPlatforms) {
			t.Errorf("Expected %d platforms, got %d", len(expectedPlatforms), len(platforms))
		}

		for i, expected := range expectedPlatforms {
			if platforms[i] != expected {
				t.Errorf("Expected platform %q at index %d, got %q", expected, i, platforms[i])
			}
		}
	})

	// Test interface implementation
	t.Run("Interface Implementation", func(t *testing.T) {
		// This test mainly checks that the interface is correctly implemented
		// We can't easily test the actual download without proper setup

		// Load config for proper initialization
		rootConfig, err := config.Load()
		if err != nil {
			t.Skipf("Skipping test due to config load error: %v", err)
			return
		}

		config := &types.DownloaderConfig{
			Level:          "standard",
			Output:         "./test_output",
			ConflictPolicy: "skip",
			Root:           rootConfig,
		}

		// Test factory creation
		factory := &downloader.NetEaseDownloaderFactory{}
		downloader, err := factory.CreateDownloader(context.Background(), config)
		if err != nil {
			t.Errorf("Unexpected error creating downloader: %v", err)
			return
		}

		// Verify downloader is not nil
		if downloader == nil {
			t.Error("Expected non-nil downloader")
		}
	})

	// Test DownloaderConfig validation
	t.Run("Config Validation", func(t *testing.T) {
		_, err := config.Load()
		if err != nil {
			t.Skipf("Skipping test due to config load error: %v", err)
			return
		}

		// This test validates the config structure without network calls
		testConfigs := []struct {
			name        string
			config      types.DownloaderConfig
			shouldError bool
			errorMsg    string
		}{
			{
				name: "valid config",
				config: types.DownloaderConfig{
					Level:          "standard",
					Output:         "./test",
					ConflictPolicy: "skip",
				},
				shouldError: false,
			},
			{
				name: "invalid conflict policy",
				config: types.DownloaderConfig{
					Level:          "standard",
					Output:         "./test",
					ConflictPolicy: "invalid",
				},
				shouldError: true,
				errorMsg:    "invalid conflict policy",
			},
		}

		for _, tc := range testConfigs {
			t.Run(tc.name, func(t *testing.T) {
				// We can't easily create an API client for testing, but we can validate the structure
				t.Logf("Testing config: %+v", tc.config)

				// Validate conflict policy separately
				_, err := types.ParseConflictPolicy(tc.config.ConflictPolicy)

				if tc.shouldError && err == nil {
					t.Errorf("Expected error for config %v", tc.config)
				}
				if !tc.shouldError && err != nil {
					t.Errorf("Unexpected error for valid config: %v", err)
				}
				if tc.shouldError && err != nil && tc.errorMsg != "" {
					if err.Error()[:len(tc.errorMsg)] != tc.errorMsg {
						t.Errorf("Expected error to start with %q, got %q", tc.errorMsg, err.Error())
					}
				}
			})
		}
	})
}
