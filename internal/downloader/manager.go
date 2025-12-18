package downloader

import (
	"context"
	"fmt"

	"github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/stkevintan/miko/internal/config"
	"github.com/stkevintan/miko/internal/models"
)

// DownloadResult represents the result of a download operation
// Downloader interface defines the contract for different music downloaders
type Downloader interface {
	// DownloadBatch downloads multiple songs and returns the batch result
	Download(ctx context.Context, music []*models.Music) (*models.BatchDownloadResponse, error)

	// GetMusic returns the music information array
	GetMusic(ctx context.Context, uris []string) ([]*models.Music, error)

	// GetLevel returns the quality level
	GetLevel() types.Level

	// GetOutput returns the output directory
	GetOutput() string

	// GetConflictPolicy returns the conflict handling policy
	GetConflictPolicy() ConflictPolicy

	Close(ctx context.Context) error
}

// DownloaderConfig represents the configuration for creating downloaders
type DownloaderConfig struct {
	Level          string
	Output         string
	ConflictPolicy string
	Root           *config.Config
}

// DownloaderFactory creates downloaders for different music platforms
type DownloaderFactory interface {
	// CreateDownloader creates a new downloader instance
	CreateDownloader(ctx context.Context, config *DownloaderConfig) (Downloader, error)

	// SupportedPlatforms returns the list of supported platforms
	SupportedPlatforms() []string
}

// NetEaseDownloaderFactory implements DownloaderFactory for NetEase Cloud Music
type NetEaseDownloaderFactory struct{}

// CreateDownloader creates a NetEase downloader
func (f *NetEaseDownloaderFactory) CreateDownloader(ctx context.Context, config *DownloaderConfig) (Downloader, error) {
	return NewNetEaseDownloader(config)
}

// SupportedPlatforms returns NetEase as the supported platform
func (f *NetEaseDownloaderFactory) SupportedPlatforms() []string {
	return []string{"netease", "163"}
}

// DownloaderManager manages multiple downloader factories
type DownloaderManager struct {
	factories map[string]DownloaderFactory
}

// NewDownloaderManager creates a new downloader manager
func NewDownloaderManager() *DownloaderManager {
	dm := &DownloaderManager{
		factories: make(map[string]DownloaderFactory),
	}

	// Register default factories
	dm.RegisterFactory("netease", &NetEaseDownloaderFactory{})
	dm.RegisterFactory("163", &NetEaseDownloaderFactory{})

	return dm
}

// RegisterFactory registers a new downloader factory
func (dm *DownloaderManager) RegisterFactory(platform string, factory DownloaderFactory) {
	dm.factories[platform] = factory
}

// CreateDownloader creates a downloader for the specified platform
func (dm *DownloaderManager) CreateDownloader(ctx context.Context, platform string, config *DownloaderConfig) (Downloader, error) {
	factory, exists := dm.factories[platform]
	if !exists {
		return nil, fmt.Errorf("unsupported platform: %s, available platforms: %v", platform, dm.GetSupportedPlatforms())
	}

	return factory.CreateDownloader(ctx, config)
}

// GetSupportedPlatforms returns all supported platforms
func (dm *DownloaderManager) GetSupportedPlatforms() []string {
	var platforms []string
	for platform := range dm.factories {
		platforms = append(platforms, platform)
	}
	return platforms
}
