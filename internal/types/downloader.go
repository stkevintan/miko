package types

import (
	"context"

	"github.com/stkevintan/miko/internal/config"
	"github.com/stkevintan/miko/internal/models"
)

type DownloaderConfig struct {
	Level          string
	Output         string
	ConflictPolicy string
	Root           *config.Config
}

// DownloadResult represents the result of a download operation
// Downloader interface defines the contract for different music downloaders
type Downloader interface {
	// DownloadBatch downloads multiple songs and returns the batch result
	Download(ctx context.Context, music []*models.Music) (*models.BatchDownloadResponse, error)

	// GetMusic returns the music information array
	GetMusic(ctx context.Context, uris []string) ([]*models.Music, error)

	// GetLevel returns the quality level
	GetLevel() string

	// GetOutput returns the output directory
	GetOutput() string

	// GetConflictPolicy returns the conflict handling policy
	GetConflictPolicy() ConflictPolicy

	Close(ctx context.Context) error
}

// DownloaderConfig represents the configuration for creating downloaders

// DownloaderFactory creates downloaders for different music platforms
type DownloaderFactory interface {
	// CreateDownloader creates a new downloader instance
	CreateDownloader(ctx context.Context, config *DownloaderConfig) (Downloader, error)

	// SupportedPlatforms returns the list of supported platforms
	SupportedPlatforms() []string
}
