package subsonic

import (
	"net/http"
	"path/filepath"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/browser"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/scanner"
	"gorm.io/gorm"
)

func (s *Subsonic) handleGetMusicFolders(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())
	cfg := di.MustInvoke[*config.Config](r.Context())

	var folders []models.MusicFolder
	// Ensure folders from config are in DB
	for _, path := range cfg.Subsonic.Folders {
		var folder models.MusicFolder
		db.Where(models.MusicFolder{Path: path}).Attrs(models.MusicFolder{Name: filepath.Base(path)}).FirstOrCreate(&folder)
		folders = append(folders, folder)
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.MusicFolders = &models.MusicFolders{
		MusicFolder: folders,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetIndexes(w http.ResponseWriter, r *http.Request) {
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	cfg := di.MustInvoke[*config.Config](r.Context())
	br := di.MustInvoke[*browser.Browser](r.Context())

	folderID, err := getQueryInt[uint](r, "musicFolderId")
	hasFolderId := err == nil

	indexes, err := br.GetIndexes(cfg.Subsonic.BrowseMode, folderID, hasFolderId, cfg.Subsonic.IgnoredArticles)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to query indexes: "+err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Indexes = &models.Indexes{
		LastModified:    sc.LastScanTime(),
		IgnoredArticles: cfg.Subsonic.IgnoredArticles,
		Index:           indexes,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetMusicDirectory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}
	cfg := di.MustInvoke[*config.Config](r.Context())
	br := di.MustInvoke[*browser.Browser](r.Context())

	dir, err := br.GetDirectory(cfg.Subsonic.BrowseMode, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			s.sendResponse(w, r, models.NewErrorResponse(70, "Directory not found"))
		} else {
			s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to query directory: "+err.Error()))
		}
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Directory = dir
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetGenres(w http.ResponseWriter, r *http.Request) {
	br := di.MustInvoke[*browser.Browser](r.Context())
	genres, err := br.GetGenres()
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to query genres"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Genres = &models.Genres{
		Genre: genres,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetArtists(w http.ResponseWriter, r *http.Request) {
	br := di.MustInvoke[*browser.Browser](r.Context())
	cfg := di.MustInvoke[*config.Config](r.Context())

	indexes, err := br.GetArtists(cfg.Subsonic.IgnoredArticles)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to query artists"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Artists = &models.ArtistsID3{
		IgnoredArticles: cfg.Subsonic.IgnoredArticles,
		Index:           indexes,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetArtist(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	br := di.MustInvoke[*browser.Browser](r.Context())

	artist, err := br.GetArtist(id)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Artist not found"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Artist = artist
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetAlbum(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	br := di.MustInvoke[*browser.Browser](r.Context())

	album, err := br.GetAlbum(id)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Album not found"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Album = album
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetSong(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	br := di.MustInvoke[*browser.Browser](r.Context())

	song, err := br.GetSong(id)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Song not found"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Song = song
	s.sendResponse(w, r, resp)
}

// TODO: Use music provider to get real data
func (s *Subsonic) handleGetArtistInfo2(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty info
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ArtistInfo2 = &models.ArtistInfo2{}
	s.sendResponse(w, r, resp)
}

// TODO: Use music provider to get real data
func (s *Subsonic) handleGetAlbumInfo2(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty info
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.AlbumInfo = &models.AlbumInfo{}
	s.sendResponse(w, r, resp)
}

// TODO: Use music provider to get real data
func (s *Subsonic) handleGetSimilarSongs2(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty list
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SimilarSongs2 = &models.SimilarSongs2{
		Song: []models.Child{},
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetAlbumInfo(w http.ResponseWriter, r *http.Request) {
	s.handleGetAlbumInfo2(w, r)
}

func (s *Subsonic) handleGetArtistInfo(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty info
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ArtistInfo = &models.ArtistInfo{}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetSimilarSongs(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty list
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SimilarSongs = &models.SimilarSongs{
		Song: []models.Child{},
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetTopSongs(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")
	if artist == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "Artist is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty list
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.TopSongs = &models.TopSongs{
		Song: []models.Child{},
	}
	s.sendResponse(w, r, resp)
}
