package netease

import (
	"context"
	"fmt"

	"github.com/chaunsin/netease-cloud-music/api"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
	nmlog "github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/registry"
)

// Initialize default logger for netease package
func init() {
	nmlog.Default = nmlog.New(&nmlog.Config{
		Level:  "info",
		Format: "json",
		Stdout: true,
	})

}

// NetEaseProviderFactory implements DownloaderFactory for NetEase Cloud Music
type NetEaseProviderFactory struct {
	cfg *api.Config
}

func NewNetEaseProviderFactory(cfg *api.Config) *NetEaseProviderFactory {
	return &NetEaseProviderFactory{
		cfg: cfg,
	}
}

// CreateProvider creates a NetEase provider
func (f *NetEaseProviderFactory) CreateProvider() (registry.Provider, error) {
	return NewProvider(f.cfg)
}

// SupportedPlatforms returns NetEase as the supported platform
func (f *NetEaseProviderFactory) SupportedPlatforms() []string {
	return []string{"netease", "163"}
}

// NMDownloader implements the NetEase Cloud Music downloader
type NMProvider struct {
	cli     *api.Client
	request *weapi.Api
}

var _ registry.Provider = (*NMProvider)(nil)

// NewProvider creates a new NMProvider for multiple songs (returns concrete type)
func NewProvider(nmApi *api.Config) (*NMProvider, error) {
	cli, err := api.NewClient(nmApi, nmlog.Default)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	request := weapi.New(cli)

	return &NMProvider{
		cli:     cli,
		request: request,
	}, nil
}

// Close closes the provider and cleans up resources
func (d *NMProvider) Close(ctx context.Context) error {
	refresh, err := d.request.TokenRefresh(ctx, &weapi.TokenRefreshReq{})
	if err != nil || refresh.Code != 200 {
		log.Warn("TokenRefresh resp:%+v err: %s", refresh, err)
	}
	return d.cli.Close(ctx)
}
