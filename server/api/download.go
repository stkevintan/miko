package api

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/provider"
	"github.com/stkevintan/miko/pkg/types"
)

// handleDownload handles music download requests
func (h *Handler) handleDownload(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	query := r.URL.Query()
	uris := query["uri"]
	level := query.Get("level")
	if level == "" {
		level = "lossless"
	}
	output := query.Get("output")
	timeoutStr := query.Get("timeout")
	conflictPolicy := query.Get("conflict_policy")
	if conflictPolicy == "" {
		conflictPolicy = "skip"
	}
	platform := query.Get("platform")
	if platform == "" {
		platform = h.cfg.Provider.Platform
	}

	// Validate required parameters
	if len(uris) == 0 {
		errorResp := models.ErrorResponse{Error: "uri query parameter is required"}
		JSON(w, http.StatusBadRequest, errorResp)
		return
	}

	// Parse timeout
	timeoutMs := 60000 // default
	if timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed >= 0 {
			timeoutMs = parsed
		}
	}

	// Convert timeout from milliseconds to duration
	timeout := time.Duration(timeoutMs) * time.Millisecond

	req := &DownloadRequest{
		URIs:     uris,
		Timeout:  timeout,
		Platform: platform,
		DownloadConfig: types.DownloadConfig{
			Level:          level,
			Output:         output,
			ConflictPolicy: conflictPolicy,
		},
	}

	ctx, err := h.getRequestInjector(r)
	if err != nil {
		JSON(w, http.StatusInternalServerError, &models.ErrorResponse{Error: err.Error()})
		return
	}

	result, err := req.Download(ctx)

	if err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		JSON(w, http.StatusInternalServerError, errorResp)
		return
	}

	if result == nil {
		errorResp := models.ErrorResponse{Error: "Download failed: no result returned"}
		JSON(w, http.StatusInternalServerError, errorResp)
		return
	}
	// Create message based on results
	var message string
	if result.Total() == 1 {
		if result.SuccessCount() == 1 {
			message = "Download URL generated successfully"
		} else {
			message = "Download failed"
		}
	} else {
		message = fmt.Sprintf("Batch download completed: %d total, %d success, %d failed", result.Total(), result.SuccessCount(), result.FailedCount())
	}

	JSON(w, http.StatusOK, &models.DownloadSummary{
		Summary: message,
		Details: result.Results(),
	})
}

// DownloadRequest represents download arguments for any resource type
type DownloadRequest struct {
	types.DownloadConfig
	Platform string   // music platform
	URIs     []string // can be song ID, URL, etc.
	Timeout  time.Duration
}

func (r *DownloadRequest) Download(ctx context.Context) (*types.MusicDownloadResults, error) {
	var (
		nctx   context.Context
		cancel context.CancelFunc
	)

	if r.Output != "" && !filepath.IsAbs(r.Output) {
		abs, err := filepath.Abs(r.Output)
		if err != nil {
			return nil, fmt.Errorf("resolve output path: %w", err)
		}
		r.Output = abs
	}

	if r.Timeout == 0 {
		nctx, cancel = context.WithCancel(ctx)
	} else {
		nctx, cancel = context.WithTimeout(ctx, r.Timeout)
	}

	defer cancel()
	provider, err := di.InvokeNamed[provider.Provider](ctx, r.Platform)
	if err != nil {
		return nil, fmt.Errorf("create provider: %w", err)
	}
	defer provider.Close(nctx)

	music, err := provider.GetMusic(nctx, r.URIs)
	if err != nil {
		return nil, fmt.Errorf("GetMusic: %w", err)
	}

	return provider.Download(nctx, music, &r.DownloadConfig)
}
