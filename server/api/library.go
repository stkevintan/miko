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
	sc := scanner.New(db, di.MustInvoke[*config.Config](r.Context()))
	if err := sc.UpdateSongMetadata(&song); err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to update metadata: " + err.Error()})
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
	sc := scanner.New(db, di.MustInvoke[*config.Config](r.Context()))
	if err := sc.SaveCoverArt(song.CoverArt, data); err != nil {
		// Log error but don't fail the request
	}

	JSON(w, http.StatusOK, song)
}

func (h *Handler) handleGetLibraryFolders(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())
	cfg := di.MustInvoke[*config.Config](r.Context())

	type FolderWithID struct {
		models.MusicFolder
		DirectoryID string `json:"directoryId"`
	}

	var folders []FolderWithID
	for _, path := range cfg.Subsonic.Folders {
		var folder models.MusicFolder
		db.Where(models.MusicFolder{Path: path}).Attrs(models.MusicFolder{Name: filepath.Base(path)}).FirstOrCreate(&folder)

		var child models.Child
		db.Select("id").Where("path = ? AND is_dir = ?", path, true).First(&child)

		folders = append(folders, FolderWithID{
			MusicFolder: folder,
			DirectoryID: child.ID,
		})
	}

	JSON(w, http.StatusOK, folders)
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
