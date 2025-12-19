package netease

import (
	"context"
	"fmt"

	"github.com/chaunsin/netease-cloud-music/api"
	nmTypes "github.com/chaunsin/netease-cloud-music/api/types"
	"github.com/chaunsin/netease-cloud-music/api/weapi"
	nmlog "github.com/chaunsin/netease-cloud-music/pkg/log"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/types"
)

// NMDownloader implements the NetEase Cloud Music downloader
type NMDownloader struct {
	cli            *api.Client
	request        *weapi.Api
	Level          nmTypes.Level
	Output         string
	ConflictPolicy types.ConflictPolicy
}

var _ types.Downloader = (*NMDownloader)(nil)

// NewNetEaseDownloader creates a new NetEase downloader that implements the Downloader interface
func NewNetEaseDownloader(config *types.DownloaderConfig) (types.Downloader, error) {
	return NewDownloader(config)
}

// NewDownloader creates a new NMDownloader for multiple songs (returns concrete type)
func NewDownloader(config *types.DownloaderConfig) (*NMDownloader, error) {
	// Validate basic config
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Validate and parse conflict policy
	policy, err := types.ParseConflictPolicy(config.ConflictPolicy)
	if err != nil {
		return nil, fmt.Errorf("invalid conflict policy: %w", err)
	}

	cli, err := api.NewClient(config.Root.NmApi, nmlog.Default)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	request := weapi.New(cli)

	// Validate and parse level
	dlevel, err := ValidateQualityLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid quality level: %w", err)
	}

	return &NMDownloader{
		cli:            cli,
		request:        request,
		Level:          dlevel,
		Output:         config.Output,
		ConflictPolicy: policy,
	}, nil
}

// Name returns the name of this downloader
func (d *NMDownloader) Name() string {
	return "netease"
}

// ValidateConfig validates the downloader configuration
func (d *NMDownloader) ValidateConfig() error {
	if d.cli == nil {
		return fmt.Errorf("API client not initialized")
	}
	if d.request == nil {
		return fmt.Errorf("request handler not initialized")
	}
	return nil
}

// GetLevel returns the quality level setting
func (d *NMDownloader) GetLevel() string {
	return string(d.Level)
}

// GetOutput returns the output directory
func (d *NMDownloader) GetOutput() string {
	return d.Output
}

// GetConflictPolicy returns the conflict policy
func (d *NMDownloader) GetConflictPolicy() types.ConflictPolicy {
	return d.ConflictPolicy
}

// Close closes the downloader and cleans up resources
func (d *NMDownloader) Close(ctx context.Context) error {
	refresh, err := d.request.TokenRefresh(ctx, &weapi.TokenRefreshReq{})
	if err != nil || refresh.Code != 200 {
		log.Warn("TokenRefresh resp:%+v err: %s", refresh, err)
	}
	return d.cli.Close(ctx)
}
