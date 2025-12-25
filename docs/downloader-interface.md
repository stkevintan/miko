# Generic Downloader Interface

This document describes the generic downloader interface implementation that abstracts different music platform downloaders.

## Overview

The generic downloader interface provides a unified way to interact with different music platform downloaders (currently NetEase Cloud Music, with support for future platforms).

## Architecture

### Interface Definition

```go
type Downloader interface {
    // Download downloads multiple songs and returns the batch result
    Download(ctx context.Context, music []*models.Music) (*models.BatchDownloadResponse, error)

    // GetMusic returns the music information array from URIs
    GetMusic(ctx context.Context, uris []string) ([]*models.Music, error)

    // GetLevel returns the quality level
    GetLevel() types.Level

    // GetOutput returns the output directory
    GetOutput() string

    // GetConflictPolicy returns the conflict handling policy
    GetConflictPolicy() ConflictPolicy

    // Close cleans up downloader resources
    Close(ctx context.Context) error
}
```

### Factory Pattern

```go
type DownloaderFactory interface {
    // CreateDownloader creates a new downloader instance
    CreateDownloader(ctx context.Context, config *DownloaderConfig) (Downloader, error)
    
    // SupportedPlatforms returns the list of supported platforms
    SupportedPlatforms() []string
}
```

### Manager

```go
type DownloaderManager struct {
    factories map[string]DownloaderFactory
}

// Methods:
func NewDownloaderManager() *DownloaderManager
func (dm *DownloaderManager) RegisterFactory(platform string, factory DownloaderFactory)
func (dm *DownloaderManager) CreateDownloader(ctx context.Context, platform string, config *DownloaderConfig) (Downloader, error)
func (dm *DownloaderManager) GetSupportedPlatforms() []string
```

## Usage

### Basic Usage

```go
// Create manager (done once in service initialization)
manager := downloader.NewDownloaderManager()

// Create a downloader instance
dl, err := manager.CreateDownloader(
    ctx,
    "netease", // platform
    &downloader.DownloaderConfig{
        Level:          "standard",
        Output:         "./downloads",
        ConflictPolicy: "skip",
        Root:           configInstance, // *config.Config
    },
)
if err != nil {
    return err
}
defer dl.Close(ctx)

// Get music from URIs (song IDs, URLs, etc.)
music, err := dl.GetMusic(ctx, []string{"123456", "https://music.163.com/song?id=789"})
if err != nil {
    return err
}

// Download the songs
result, err := dl.Download(ctx, music)
// result is *models.BatchDownloadResponse
```

### Configuration

```go
type DownloaderConfig struct {
    Level          string        // quality level ("standard", "higher", "lossless", etc.)
    Output         string        // output directory path
    ConflictPolicy string        // conflict policy ("skip", "overwrite", "rename", "update_tags")
    Root           *config.Config // application configuration
}
```

### Service Layer Integration

```go
type DownloadOptions struct {
    Platform       string        // music platform ("netease")
    URIs           []string      // song IDs, URLs, etc.
    Level          string        // quality level
    Output         string        // output directory
    Timeout        time.Duration // download timeout
    ConflictPolicy string        // conflict policy
}
```

## Supported Platforms

- **NetEase Cloud Music**: `"netease"`, `"163"`

## ConflictPolicy Enum

The interface uses a type-safe enum for conflict policies:

```go
type ConflictPolicy string

const (
    ConflictPolicySkip       ConflictPolicy = "skip"
    ConflictPolicyOverwrite  ConflictPolicy = "overwrite"
    ConflictPolicyRename     ConflictPolicy = "rename"
    ConflictPolicyUpdateTags ConflictPolicy = "update_tags"
)
```

## Adding New Platforms

To add support for a new music platform:

1. **Implement the Downloader interface**:
   ```go
   type YourPlatformDownloader struct {
       Level          types.Level
       Output         string
       ConflictPolicy ConflictPolicy
       // platform-specific API client, config, etc.
   }
   
   func (d *YourPlatformDownloader) Download(ctx context.Context, music []*models.Music) (*models.BatchDownloadResponse, error) {
       // implementation for batch downloading
   }
   
   func (d *YourPlatformDownloader) GetMusic(ctx context.Context, uris []string) ([]*models.Music, error) {
       // implementation to resolve URIs to Music objects
   }
   
   func (d *YourPlatformDownloader) GetLevel() types.Level {
       return d.Level
   }
   
   func (d *YourPlatformDownloader) GetOutput() string {
       return d.Output
   }
   
   func (d *YourPlatformDownloader) GetConflictPolicy() ConflictPolicy {
       return d.ConflictPolicy
   }
   
   func (d *YourPlatformDownloader) Close(ctx context.Context) error {
       // cleanup resources
   }
   ```

2. **Create a factory**:
   ```go
   type YourPlatformFactory struct{}
   
   func (f *YourPlatformFactory) CreateDownloader(ctx context.Context, config *DownloaderConfig) (Downloader, error) {
       // Validate configuration
       if config.Root == nil {
           return nil, fmt.Errorf("config.Root cannot be nil")
       }
       
       // Parse and validate quality level
       level, err := parseQualityLevel(config.Level)
       if err != nil {
           return nil, fmt.Errorf("invalid quality level: %w", err)
       }
       
       // Parse and validate conflict policy
       policy, err := ParseConflictPolicy(config.ConflictPolicy)
       if err != nil {
           return nil, fmt.Errorf("invalid conflict policy: %w", err)
       }
       
       // Create platform-specific downloader
       return &YourPlatformDownloader{
           Level:          level,
           Output:         config.Output,
           ConflictPolicy: policy,
           // Initialize platform-specific fields
       }, nil
   }
   
   func (f *YourPlatformFactory) SupportedPlatforms() []string {
       return []string{"yourplatform", "your-alias"}
   }
   ```

3. **Register the factory**:
   ```go
   // In NewDownloaderManager() or during initialization
   manager.RegisterFactory("yourplatform", &YourPlatformFactory{})
   ```

## Benefits

1. **Extensibility**: Easy to add support for new music platforms
2. **Type Safety**: Enum-based conflict policy with validation
3. **Consistent Interface**: All downloaders implement the same interface
4. **Factory Pattern**: Clean creation and platform selection
5. **Testing**: Easy to mock and test different implementations

## Migration

The service layer has been updated to use the generic interface with batch download support and URI-based input instead of single Music objects.

**Old usage:**
```go
// Old single download approach
downloader, err := downloader.NewDownloader(cli, music, level, output, policy)
result, err := downloader.Download(ctx)
```

**New usage:**
```go
// New batch download approach
dl, err := service.downloaderManager.CreateDownloader(ctx, "netease", &downloader.DownloaderConfig{
    Level:          "standard",
    Output:         "./downloads", 
    ConflictPolicy: "skip",
    Root:           config,
})
music, err := dl.GetMusic(ctx, []string{"123456", "https://music.163.com/song?id=789"})
result, err := dl.Download(ctx, music) // Returns *models.BatchDownloadResponse
```

**Service layer usage:**
```go
// Through service layer (recommended)
result, err := service.Download(ctx, &DownloadOptions{
    Platform:       "netease",
    URIs:           []string{"123456", "https://music.163.com/song?id=789"},
    Level:          "standard",
    Output:         "./downloads",
    ConflictPolicy: "skip",
    Timeout:        30 * time.Second,
})
```