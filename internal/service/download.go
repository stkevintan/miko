package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/stkevintan/miko/pkg/types"
)

// DownloadOptions represents download arguments for any resource type
type DownloadOptions struct {
	Platform       string   // music platform
	URIs           []string // can be song ID, URL, etc.
	Level          string   // quality level
	Output         string   // output directory
	Timeout        time.Duration
	ConflictPolicy string // conflict policy: skip, overwrite, rename, update_tags.
}

// Download handles downloading from various resource types (ID, URL, etc.)
func (s *Service) Download(ctx context.Context, c *DownloadOptions) (*types.MusicDownloadResults, error) {
	var (
		nctx   context.Context
		cancel context.CancelFunc
	)

	if c.Output != "" && !filepath.IsAbs(c.Output) {
		abs, err := filepath.Abs(c.Output)
		if err != nil {
			return nil, fmt.Errorf("resolve output path: %w", err)
		}
		c.Output = abs
	}

	if c.Timeout == 0 {
		nctx, cancel = context.WithCancel(ctx)
	} else {
		nctx, cancel = context.WithTimeout(ctx, c.Timeout)
	}

	if c.Platform == "" {
		c.Platform = "netease"
	}

	defer cancel()
	// Create batch downloader using the new interface
	dl, err := s.downloaderManager.CreateDownloader(
		nctx,
		c.Platform,
		&types.DownloaderConfig{
			Level:          c.Level,
			Output:         c.Output,
			ConflictPolicy: c.ConflictPolicy,
			Root:           s.config,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create batch downloader: %w", err)
	}
	defer dl.Close(nctx)

	musics, err := dl.GetMusic(nctx, c.URIs)
	if err != nil {
		return nil, fmt.Errorf("GetMusic: %w", err)
	}

	return dl.Download(nctx, musics)
}
