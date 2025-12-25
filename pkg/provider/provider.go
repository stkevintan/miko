package provider

import (
	"context"

	"github.com/stkevintan/miko/pkg/cookiecloud"
	"github.com/stkevintan/miko/pkg/types"
)

type Config struct {
	Platform string `json:"platform" mapstructure:"platform"`
}

// Provider interface defines the contract for different music providers
type Provider interface {
	GetCookieJar() cookiecloud.CookieJar

	User(ctx context.Context) (*types.User, error)

	// DownloadBatch downloads multiple songs and returns the batch result
	Download(ctx context.Context, music []*types.Music, config *types.DownloadConfig) (*types.MusicDownloadResults, error)

	// GetMusic returns the music information array
	GetMusic(ctx context.Context, uris []string) ([]*types.Music, error)

	Close(ctx context.Context) error
}
