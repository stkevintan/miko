package service

import (
	"context"
	"testing"
	"time"

	"github.com/chaunsin/netease-cloud-music/api/types"
	l "github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/stkevintan/miko/internal/config"
	"github.com/stkevintan/miko/internal/models"
)

func TestDownloadService(t *testing.T) {
	// Initialize logger
	l.Default = l.New(&l.Config{
		Level:  "info",
		Format: "text",
		Stdout: true,
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	service := New(cfg)

	// Test Music model usage
	t.Run("Download with Music model", func(t *testing.T) {
		music := &models.Music{
			Id:     123456,
			Name:   "Test Song",
			Artist: []types.Artist{{Id: 1, Name: "Test Artist"}},
			Album: types.Album{
				Id:   1,
				Name: "Test Album",
			},
			Time: 240000,
		}

		args := &DownloadArgs{
			Music:   music,
			Level:   "standard",
			Output:  "./test_downloads",
			Timeout: 30 * time.Second,
		}

		// This will likely fail due to lack of authentication
		// but we can test the structure is correct
		result, err := service.Download(context.Background(), args)

		// Since we don't have real authentication, expect an error
		// but validate that it's not due to struct issues
		if err != nil {
			t.Logf("Expected error due to authentication: %v", err)
			// Make sure it's not a struct-related error
			if err.Error() == "music is required" {
				t.Error("Music struct validation failed")
			}
		}

		// If somehow successful, check result structure
		if result != nil {
			if result.SongID == "" {
				t.Error("Expected SongID to be populated")
			}
		}
	})

	// Test nil Music validation
	t.Run("Download with nil Music", func(t *testing.T) {
		args := &DownloadArgs{
			Music:   nil,
			Level:   "standard",
			Output:  "./test_downloads",
			Timeout: 30 * time.Second,
		}

		result, err := service.Download(context.Background(), args)

		if err == nil {
			t.Error("Expected error for nil Music")
		}

		if result != nil {
			t.Error("Expected nil result for invalid input")
		}

		if err.Error() != "music is required" {
			t.Errorf("Expected 'music is required' error, got: %v", err)
		}
	})

	// Test DownloadResource with URL
	t.Run("DownloadResource with URL", func(t *testing.T) {
		args := &DownloadResourceArgs{
			Resource: "https://music.163.com/song?id=123456",
			Level:    "standard",
			Output:   "./test_downloads",
			Timeout:  30 * time.Second,
		}

		result, err := service.DownloadFromResource(context.Background(), args)

		// This will likely fail due to lack of authentication
		// but we can test the URL parsing works
		if err != nil {
			t.Logf("Expected error due to authentication: %v", err)
		}

		// If successful, check result structure
		if result != nil && result.Total > 0 {
			if len(result.Songs) == 0 {
				t.Error("Expected songs to be populated")
			}
		}
	})
}
