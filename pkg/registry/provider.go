package registry

import (
	"context"

	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/types"
)

// Provider interface defines the contract for different music providers
type Provider interface {
	GetCookieJar() cookiecloud.CookieJar

	Auth(ctx context.Context, key string, password string) (*types.LoginResult, error)

	// DownloadBatch downloads multiple songs and returns the batch result
	Download(ctx context.Context, music []*types.Music, config *types.DownloadConfig) (*types.MusicDownloadResults, error)

	// GetMusic returns the music information array
	GetMusic(ctx context.Context, uris []string) ([]*types.Music, error)

	Close(ctx context.Context) error
}

// ProviderFactory creates music provider for different music platforms
type ProviderFactory interface {
	CreateProvider() (Provider, error)
}
