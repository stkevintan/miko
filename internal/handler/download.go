package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/internal/models"
	"github.com/stkevintan/miko/internal/service"
)

// handleDownload handles music download requests
// @Summary      Download music tracks
// @Description  Download single or multiple music tracks by providing song IDs, album URLs, playlist URLs, or song URLs. Supports batch downloads with various quality levels and conflict resolution policies.
// @Tags         music
// @Accept       json
// @Produce      json
// @Param        uri query []string true "Resource URIs to download (can be song ID, album URL, playlist URL, song URL). Multiple URIs supported for batch downloads." example("2161154646") example("https://music.163.com/song?id=2161154646")
// @Param        level query string false "Audio quality level" example("lossless") Enums(standard, higher, exhigh, lossless, hires, 128, 192, 320, HQ, SQ, HR) default("lossless")
// @Param        output query string false "Output directory path for downloaded files" example("./downloads")
// @Param        timeout query int false "Timeout in milliseconds for the download operation" example(60000)
// @Param        conflict_policy query string false "How to handle existing files" example("skip") Enums(skip, overwrite, rename, update_tags) default("skip")
// @Success      200 {object} models.DownloadSummary "Successful batch download response with individual song results and error details"
// @Failure      400 {object} models.ErrorResponse "Bad request - missing or invalid parameters"
// @Failure      500 {object} models.ErrorResponse "Internal server error during download processing"
// @Router       /download [get]
func (h *Handler) handleDownload(c *gin.Context) {
	// Get query parameters
	uris := c.QueryArray("uri")
	level := c.DefaultQuery("level", "lossless")
	output := c.DefaultQuery("output", "")
	timeoutStr := c.Query("timeout")
	conflictPolicy := c.DefaultQuery("conflict_policy", "skip")
	platform := c.DefaultQuery("platform", "netease")

	// Validate required parameters
	if len(uris) == 0 {
		errorResp := models.ErrorResponse{Error: "uri query parameter is required"}
		c.JSON(http.StatusBadRequest, errorResp)
		return
	}

	// Parse timeout
	timeoutMs := 60000 // default
	if timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed > 0 {
			timeoutMs = parsed
		}
	}

	// Convert timeout from milliseconds to duration
	timeout := time.Duration(timeoutMs) * time.Millisecond

	result, err := h.service.Download(c.Request.Context(), &service.DownloadOptions{
		URIs:           uris,
		Level:          level,
		Output:         output,
		Timeout:        timeout,
		ConflictPolicy: conflictPolicy,
		Platform:       platform,
	})

	if err != nil {
		errorResp := models.ErrorResponse{Error: err.Error()}
		c.JSON(http.StatusInternalServerError, errorResp)
		return
	}

	if result == nil {
		errorResp := models.ErrorResponse{Error: "Download failed: no result returned"}
		c.JSON(http.StatusInternalServerError, errorResp)
		return
	}
	// Create message based on results
	var message string
	if result.Total == 1 {
		if result.Success == 1 {
			message = "Download URL generated successfully"
		} else {
			message = "Download failed"
		}
	} else {
		message = fmt.Sprintf("Batch download completed: %d total, %d success, %d failed", result.Total, result.Success, result.Failed)
	}

	c.JSON(http.StatusOK, &models.DownloadSummary{
		Summary: message,
		Details: result,
	})
}
