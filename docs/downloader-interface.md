# Generic Provider Interface

This document describes the generic music provider interface implementation that abstracts different music platform implementations.

## Overview

The generic provider interface provides a unified way to interact with different music platforms (currently NetEase Cloud Music, with support for future platforms). It leverages Dependency Injection for resource management and request-scoped isolation.

## Architecture

### Interface Definition

```go
type Provider interface {
    // GetCookieJar returns the CookieJar used by the provider
    GetCookieJar() cookiecloud.CookieJar

    // User retrieves user information from the platform
    User(ctx context.Context) (*types.User, error)

    // Download downloads multiple songs and returns the batch result
    Download(ctx context.Context, music []*types.Music, config *types.DownloadConfig) (*types.MusicDownloadResults, error)

    // GetMusic returns the music information array from URIs
    GetMusic(ctx context.Context, uris []string) ([]*types.Music, error)

    // Close cleans up provider resources
    Close(ctx context.Context) error
}
```

### Dependency Injection

The project uses `samber/do/v2` for dependency injection. Providers are registered as named services and resolved within a request-scoped injector that provides a user-specific `CookieJar`.

```go
// Example registration in a request scope
do.ProvideNamed(scope, "netease", netease.NewNetEaseProvider)

// Example invocation
provider, err := do.InvokeNamed[provider.Provider](injector, "netease")
```

## Usage

### Basic Usage

```go
// Resolve a provider from the injector (usually done in an API handler)
p, err := do.InvokeNamed[provider.Provider](injector, "netease")
if err != nil {
    return err
}
defer p.Close(ctx)

// Get music from URIs (song IDs, URLs, etc.)
music, err := p.GetMusic(ctx, []string{"2161154646", "https://music.163.com/song?id=441532"})
if err != nil {
    return err
}

// Download the songs
result, err := p.Download(ctx, music, &types.DownloadConfig{
    Level:          "lossless",
    Output:         "./songs",
    ConflictPolicy: "skip",
})
```

### Cookie Management

Each provider is initialized with a `CookieJar` that is automatically synchronized with a CookieCloud server. This allows the provider to maintain authenticated sessions across different environments.
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