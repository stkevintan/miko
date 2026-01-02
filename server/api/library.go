package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/browser"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/musicbrainz"
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
	sc.ScanPath(r.Context(), song.Path)

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

func (h *Handler) handleScanLibrary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	if req.ID == "" {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ID is required"})
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var item models.Child
	if err := db.Where("id = ?", req.ID).First(&item).Error; err != nil {
		JSON(w, http.StatusNotFound, models.ErrorResponse{Error: "Item not found"})
		return
	}

	sc := scanner.New(db, di.MustInvoke[*config.Config](r.Context()))
	sc.ScanPath(r.Context(), item.Path)

	// Fetch updated item
	if err := db.Where("id = ?", req.ID).First(&item).Error; err == nil {
		JSON(w, http.StatusOK, item)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleScanAllLibrary(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())
	cfg := di.MustInvoke[*config.Config](r.Context())
	sc := scanner.New(db, cfg)

	// Use background context or app context if available to ensure scan continues
	// For now, we'll just run it in a goroutine.
	// In a real app, you'd want to manage this more carefully.
	go sc.ScanAll(h.ctx, false)

	JSON(w, http.StatusOK, map[string]string{"status": "scanning"})
}

func (h *Handler) handleGetScanStatus(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())
	cfg := di.MustInvoke[*config.Config](r.Context())
	sc := scanner.New(db, cfg)

	var count int64
	db.Model(&models.Child{}).Where("is_dir = ?", false).Count(&count)

	JSON(w, http.StatusOK, map[string]interface{}{
		"scanning": sc.IsScanning(),
		"count":    count,
	})
}

func (h *Handler) handleScrapeLibrarySongs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
		ID  string   `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	ids := req.IDs
	if req.ID != "" {
		ids = append(ids, req.ID)
	}

	if len(ids) == 0 {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ID or IDs are required"})
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	cfg := di.MustInvoke[*config.Config](r.Context())
	mb := musicbrainz.NewClient("Miko/1.0.0 (https://github.com/stkevintan/miko)")
	sc := scanner.New(db, cfg)

	var songs []models.Child
	if err := db.Where("id IN ? AND is_dir = ?", ids, false).Find(&songs).Error; err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to find songs: " + err.Error()})
		return
	}

	for i := range songs {
		_ = h.scrapeSong(r.Context(), db, mb, sc, &songs[i])
	}

	if req.ID != "" && len(songs) > 0 {
		JSON(w, http.StatusOK, songs[0])
	} else {
		JSON(w, http.StatusOK, songs)
	}
}

func (h *Handler) scrapeSong(ctx context.Context, db *gorm.DB, mb *musicbrainz.Client, sc *scanner.Scanner, song *models.Child) error {
	searchResult, err := mb.SearchRecording(song.Artist, song.Album, song.Title)
	if err != nil {
		return fmt.Errorf("failed to search on MusicBrainz: %w", err)
	}

	// Get full details
	recording, err := mb.GetRecording(searchResult.ID)
	if err != nil {
		// Fallback to search result if lookup fails
		recording = searchResult
	}

	// Prepare tags for update
	newTags := map[string][]string{
		tags.Title:              {recording.Title},
		tags.MusicBrainzTrackID: {recording.ID},
	}

	if len(recording.Artists) > 0 {
		artists := make([]string, len(recording.Artists))
		artistIDs := make([]string, len(recording.Artists))
		for i, a := range recording.Artists {
			artists[i] = a.Name
			artistIDs[i] = a.ID
		}
		newTags[tags.Artists] = artists
		newTags[tags.MusicBrainzArtistID] = artistIDs
	}

	if len(recording.ISRCs) > 0 {
		newTags[tags.ISRC] = recording.ISRCs
	}

	// Genres and Tags
	genres := make([]string, 0)
	for _, g := range recording.Genres {
		genres = append(genres, g.Name)
	}
	for _, t := range recording.Tags {
		genres = append(genres, t.Name)
	}
	if len(genres) > 0 {
		newTags[tags.Genre] = genres
	}

	// Relationships (Composer, Lyricist, etc.)
	addUniqueTag := func(key string, value string) {
		if value == "" {
			return
		}
		for _, v := range newTags[key] {
			if v == value {
				return
			}
		}
		newTags[key] = append(newTags[key], value)
	}

	for _, rel := range recording.Relations {
		switch rel.Type {
		case "composer":
			addUniqueTag(tags.Composer, rel.Artist.Name)
		case "lyricist":
			addUniqueTag(tags.Lyricist, rel.Artist.Name)
		case "producer":
			addUniqueTag(tags.Producer, rel.Artist.Name)
		}

		// Check work relations for composer/lyricist if not found on recording
		if rel.Work.Title != "" {
			for _, wrel := range rel.Work.Relations {
				switch wrel.Type {
				case "composer":
					addUniqueTag(tags.Composer, wrel.Artist.Name)
				case "lyricist":
					addUniqueTag(tags.Lyricist, wrel.Artist.Name)
				}
			}
		}
	}

	if len(recording.Releases) > 0 {
		release := recording.Releases[0]
		// Try to find a release that matches the album name
		if song.Album != "" {
			for _, r := range recording.Releases {
				if strings.EqualFold(r.Title, song.Album) {
					release = r
					break
				}
			}
		}

		newTags[tags.Album] = []string{release.Title}
		newTags[tags.MusicBrainzAlbumID] = []string{release.ID}
		newTags[tags.MusicBrainzReleaseGroupID] = []string{release.ReleaseGroup.ID}

		if release.Status != "" {
			newTags[tags.ReleaseStatus] = []string{release.Status}
		}
		if release.Country != "" {
			newTags[tags.ReleaseCountry] = []string{release.Country}
		}
		if release.Barcode != "" {
			newTags[tags.Barcode] = []string{release.Barcode}
		}
		if release.ReleaseGroup.Type != "" {
			newTags[tags.ReleaseType] = []string{release.ReleaseGroup.Type}
		}

		if len(release.ArtistCredit) > 0 {
			albumArtists := make([]string, len(release.ArtistCredit))
			albumArtistIDs := make([]string, len(release.ArtistCredit))
			for i, a := range release.ArtistCredit {
				albumArtists[i] = a.Name
				albumArtistIDs[i] = a.ID
			}
			newTags[tags.AlbumArtist] = albumArtists
			newTags[tags.MusicBrainzAlbumArtistID] = albumArtistIDs
		}

		if release.Date != "" {
			newTags[tags.Date] = []string{release.Date}
		}

		if len(release.Media) > 0 {
			media := release.Media[0]
			newTags[tags.DiscNumber] = []string{fmt.Sprintf("%d", media.Position)}
			if media.Format != "" {
				newTags[tags.Media] = []string{media.Format}
			}
			if len(media.Track) > 0 {
				newTags[tags.TrackNumber] = []string{media.Track[0].Number}
			}
		}
	}

	// Update tags in file
	if err := tags.Write(song.Path, newTags); err != nil {
		return fmt.Errorf("failed to write tags to file: %w", err)
	}

	// Update database using scanner logic
	sc.ScanPath(ctx, song.Path)

	// Fetch updated song
	if err := db.Where("id = ?", song.ID).First(song).Error; err != nil {
		return fmt.Errorf("failed to fetch updated song: %w", err)
	}

	return nil
}

func (h *Handler) handleDeleteLibraryItems(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
		ID  string   `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request body"})
		return
	}

	ids := req.IDs
	if req.ID != "" {
		ids = append(ids, req.ID)
	}

	if len(ids) == 0 {
		JSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "ID or IDs are required"})
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var items []models.Child
	if err := db.Where("id IN ?", ids).Find(&items).Error; err != nil {
		JSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to find items: " + err.Error()})
		return
	}

	for _, item := range items {
		// Delete from disk
		if err := os.RemoveAll(item.Path); err != nil {
			// Log error but continue with others
			fmt.Printf("Failed to delete %s from disk: %v\n", item.Path, err)
		}

		// Delete from database
		if item.IsDir {
			db.Where("path LIKE ?", item.Path+"%").Delete(&models.Child{})
		} else {
			db.Delete(&item)
		}
	}

	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
