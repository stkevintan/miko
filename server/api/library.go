package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/browser"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/scanner"
	"github.com/stkevintan/miko/pkg/scraper"
	"github.com/stkevintan/miko/pkg/tags"
	"gorm.io/gorm"
)

type UpdateSongRequest struct {
	ID   string              `json:"id"`
	Tags map[string][]string `json:"tags"`
}

func (h *Handler) handleGetLibrarySongTags(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ID is required"})
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var song models.Child
	if err := db.Select("id, path").Where("id = ?", id).First(&song).Error; err != nil {
		JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "Song not found"})
		return
	}

	allTags, err := tags.ReadAll(song.Path)
	if err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to read tags: " + err.Error()})
		return
	}

	JSON(w, http.StatusOK, allTags)
}

func (h *Handler) handleUpdateLibrarySong(w http.ResponseWriter, r *http.Request) {
	var req UpdateSongRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.ID == "" {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ID is required"})
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var song models.Child
	if err := db.Where("id = ?", req.ID).First(&song).Error; err != nil {
		JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "Song not found"})
		return
	}

	// Update tags in file
	if err := tags.Write(song.Path, req.Tags); err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to write tags to file: " + err.Error()})
		return
	}

	// Update database using scanner logic
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	sc.ScanPath(r.Context(), song.ID)

	// Fetch updated song
	if err := db.Where("id = ?", req.ID).First(&song).Error; err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to fetch updated song"})
		return
	}

	JSON(w, http.StatusOK, song)
}

func (h *Handler) handleUpdateLibrarySongCover(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Failed to parse form"})
		return
	}

	id := r.FormValue("id")
	if id == "" {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ID is required"})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "File is required"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to read file"})
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var song models.Child
	if err := db.Where("id = ?", id).First(&song).Error; err != nil {
		JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "Song not found"})
		return
	}

	// Write to file
	if err := tags.WriteImage(song.Path, data); err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to write image to file: " + err.Error()})
		return
	}

	// Update cache using scanner logic
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	if err := sc.SaveCoverArt(song.CoverArt, data); err != nil {
		// Log error but don't fail the request
	}

	JSON(w, http.StatusOK, song)
}

func (h *Handler) handleGetLibraryFolders(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())

	type FolderWithID struct {
		models.MusicFolder
		DirectoryID string `json:"directoryId"`
	}

	var folders []models.MusicFolder
	if err := db.Find(&folders).Error; err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to fetch folders"})
		return
	}

	var result []FolderWithID
	for _, folder := range folders {
		var child models.Child
		db.Select("id, is_dir").Where("path = ? AND is_dir = ?", folder.Path, true).First(&child)
		result = append(result, FolderWithID{
			MusicFolder: folder,
			DirectoryID: child.ID,
		})
	}

	JSON(w, http.StatusOK, result)
}

func (h *Handler) handleGetLibraryDirectory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ID is required"})
		return
	}
	br := di.MustInvoke[*browser.Browser](r.Context())

	dir, err := br.GetDirectory("file", id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "Directory not found"})
		} else {
			JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to query directory: " + err.Error()})
		}
		return
	}

	JSON(w, http.StatusOK, dir)
}

func (h *Handler) handleGetLibraryCoverArt(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ID is required"})
		return
	}

	coverArt := ""
	if strings.HasPrefix(id, "al-") || strings.HasPrefix(id, "ar-") {
		coverArt = id
	} else {
		db := di.MustInvoke[*gorm.DB](r.Context())
		var child models.Child
		if err := db.Model(&models.Child{}).Select("id, cover_art").Where("id = ?", id).First(&child).Error; err != nil {
			JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "Cover art not found"})
			return
		}
		coverArt = child.CoverArt
	}

	if coverArt == "" {
		JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "Cover art not found"})
		return
	}

	cfg := di.MustInvoke[*config.Config](r.Context())
	cacheDir := scanner.GetCoverCacheDir(cfg)
	cachePath := filepath.Join(cacheDir, coverArt)

	if _, err := os.Stat(cachePath); err != nil {
		JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "Cover art file not found"})
		return
	}

	http.ServeFile(w, r, cachePath)
}

func (h *Handler) handleScanLibrary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	if len(req.IDs) == 0 {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "IDs are required"})
		return
	}

	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	updatedIds := make([]string, 0, len(req.IDs))

	for _, id := range req.IDs {
		if seenIds, err := sc.ScanPath(r.Context(), id); err == nil {
			seenIds.Range(func(key, value any) bool {
				songID := key.(string)
				updatedIds = append(updatedIds, songID)
				return true
			})
		}
	}

	JSON(w, http.StatusOK, updatedIds)
}

func (h *Handler) handleScanAllLibrary(w http.ResponseWriter, r *http.Request) {
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	incremental := r.URL.Query().Get("incremental") == "true"

	// Use background context or app context if available to ensure scan continues
	// For now, we'll just run it in a goroutine.
	// In a real app, you'd want to manage this more carefully.
	go sc.ScanAll(h.ctx, incremental)

	JSON(w, http.StatusOK, map[string]string{"status": "scanning"})
}

func (h *Handler) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	sp := di.MustInvoke[*scraper.Scraper](r.Context())

	var count int64
	db.Model(&models.Child{}).Where("is_dir = ?", false).Count(&count)

	JSON(w, http.StatusOK, map[string]interface{}{
		"scanning": sc.IsScanning(),
		"scraping": sp.IsScraping(),
		"count":    count,
	})
}

func (h *Handler) handleScrapeAllLibrarySongs(w http.ResponseWriter, r *http.Request) {
	sp := di.MustInvoke[*scraper.Scraper](r.Context())

	go sp.ScrapeAll(h.ctx)

	JSON(w, http.StatusOK, map[string]string{"status": "scraping"})
}

func (h *Handler) handleScrapeLibrarySongs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	ids := req.IDs

	if len(ids) == 0 {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ids are required"})
		return
	}

	sp := di.MustInvoke[*scraper.Scraper](r.Context())
	updatedIds := make([]string, 0, len(ids))

	for _, id := range ids {
		// scrap then scan to update DB
		if seenIds, err := sp.ScrapePath(r.Context(), id); err == nil {
			seenIds.Range(func(key, value any) bool {
				songID := key.(string)
				updatedIds = append(updatedIds, songID)
				return true
			})
		}
	}

	JSON(w, http.StatusOK, updatedIds)
}

func (h *Handler) handleGetLibrarySong(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ID is required"})
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var song models.Child
	if err := db.Where("id = ?", id).First(&song).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "Song not found"})
		} else {
			JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to fetch song: " + err.Error()})
		}
		return
	}

	JSON(w, http.StatusOK, song)
}
