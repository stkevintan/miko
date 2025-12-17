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
// @Summary      Download music
// @Description  Get download URL and metadata for a song or batch download from playlist/album
// @Tags         music
// @Accept       json
// @Produce      json
// @Param        resource query string true "Resource to download (song ID, album URL, playlist URL, song URL)" example("2161154646")
// @Param        level query string false "Audio quality level" example("lossless") Enums(standard, higher, exhigh, lossless, hires, 128, 192, 320, HQ, SQ, HR)
// @Param        output query string false "Output directory path" example("./downloads")
// @Param        timeout query int false "Timeout in milliseconds" example(30000)
// @Success      200 {object} models.BatchDownloadResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      500 {object} models.ErrorResponse
// @Router       /download [get]
func (h *Handler) handleDownload(c *gin.Context) {
	// Get query parameters
	resource := c.Query("resource")
	level := c.DefaultQuery("level", "lossless")
	output := c.DefaultQuery("output", "./downloads")
	timeoutStr := c.DefaultQuery("timeout", "30000")

	// Validate required parameters
	if resource == "" {
		errorResp := models.ErrorResponse{Error: "resource query parameter is required"}
		c.JSON(http.StatusBadRequest, errorResp)
		return
	}

	// Parse timeout
	timeoutMs := 30000 // default
	if timeoutStr != "" {
		if parsed, err := strconv.Atoi(timeoutStr); err == nil && parsed > 0 {
			timeoutMs = parsed
		}
	}

	// Convert timeout from milliseconds to duration
	timeout := time.Duration(timeoutMs) * time.Millisecond

	result, err := h.service.DownloadFromResource(c.Request.Context(), &service.DownloadResourceArgs{
		Resource: resource,
		Level:    level,
		Output:   output,
		Timeout:  timeout,
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

	// Convert service result to response model
	var songs []models.DownloadResponse
	for _, song := range result.Songs {
		songs = append(songs, models.DownloadResponse{
			SongID:         song.SongID,
			SongName:       song.SongName,
			Artist:         song.Artist,
			Album:          song.Album,
			DownloadURL:    song.DownloadURL,
			DownloadedPath: song.DownloadedPath,
			Quality:        song.Quality,
			FileType:       song.FileType,
			FileSize:       song.FileSize,
			Duration:       song.Duration,
			Success:        true,
			Message:        "Download URL generated successfully",
		})
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

	response := models.BatchDownloadResponse{
		Total:   result.Total,
		Success: result.Success,
		Failed:  result.Failed,
		Songs:   songs,
		Message: message,
	}

	c.JSON(http.StatusOK, response)
}
