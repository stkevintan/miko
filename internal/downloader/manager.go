package downloader

import (
	"context"
	"fmt"

	"github.com/stkevintan/miko/internal/downloader/netease"
	"github.com/stkevintan/miko/internal/types"
)

// NetEaseDownloaderFactory implements DownloaderFactory for NetEase Cloud Music
type NetEaseDownloaderFactory struct{}

// CreateDownloader creates a NetEase downloader
func (f *NetEaseDownloaderFactory) CreateDownloader(ctx context.Context, config *types.DownloaderConfig) (types.Downloader, error) {
	return netease.NewNetEaseDownloader(config)
}

// SupportedPlatforms returns NetEase as the supported platform
func (f *NetEaseDownloaderFactory) SupportedPlatforms() []string {
	return []string{"netease", "163"}
}

// DownloaderManager manages multiple downloader factories
type DownloaderManager struct {
	factories map[string]types.DownloaderFactory
}

// NewDownloaderManager creates a new downloader manager
func NewDownloaderManager() *DownloaderManager {
	dm := &DownloaderManager{
		factories: make(map[string]types.DownloaderFactory),
	}

	// Register default factories
	dm.RegisterFactory("netease", &NetEaseDownloaderFactory{})
	dm.RegisterFactory("163", &NetEaseDownloaderFactory{})

	return dm
}

// RegisterFactory registers a new downloader factory
func (dm *DownloaderManager) RegisterFactory(platform string, factory types.DownloaderFactory) {
	dm.factories[platform] = factory
}

// CreateDownloader creates a downloader for the specified platform
func (dm *DownloaderManager) CreateDownloader(ctx context.Context, platform string, config *types.DownloaderConfig) (types.Downloader, error) {
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
