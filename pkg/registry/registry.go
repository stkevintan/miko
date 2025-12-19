package registry

import (
	"fmt"
)

type Config struct {
	DefaultPlatform string `json:"default" mapstructure:"default"`
}

// ProviderRegistry manages multiple downloader factories
type ProviderRegistry struct {
	factories map[string]ProviderFactory
	Config    *Config
}

func NewProviderRegistry(cfg *Config) (*ProviderRegistry, error) {
	return &ProviderRegistry{
		factories: make(map[string]ProviderFactory),
		Config:    cfg,
	}, nil
}

// RegisterFactory registers a new downloader factory
func (dm *ProviderRegistry) RegisterFactory(platform string, factory ProviderFactory) {
	dm.factories[platform] = factory
}

// CreateProvider creates a provider for the specified platform
func (dm *ProviderRegistry) CreateProvider(platform string) (Provider, error) {
	if platform == "" {
		platform = dm.Config.DefaultPlatform
	}
	factory, exists := dm.factories[platform]
	if !exists {
		return nil, fmt.Errorf("unsupported platform: %s, available platforms: %v", platform, dm.GetSupportedPlatforms())
	}

	return factory.CreateProvider()
}

// GetSupportedPlatforms returns all supported platforms
func (dm *ProviderRegistry) GetSupportedPlatforms() []string {
	var platforms []string
	for platform := range dm.factories {
		platforms = append(platforms, platform)
	}
	return platforms
}
