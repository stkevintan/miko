package subsonic

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
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
	db := di.MustInvoke[*gorm.DB](r.Context())
	sc := di.MustInvoke[*scanner.Scanner](r.Context())
	cfg := di.MustInvoke[*config.Config](r.Context())

	indexMap := make(map[string][]models.Artist)
	folderID, err := getQueryInt[uint](r, "musicFolderId")
	hasFolderId := err == nil

	if cfg.Subsonic.BrowseMode == "tag" {
		var artists []models.ArtistID3
		query := db.Model(&models.ArtistID3{})
		query = query.Joins("JOIN song_artists ON song_artists.artist_id3_id = artist_id3.id").
			Joins("JOIN children ON children.id = song_artists.child_id")
		if hasFolderId {
			query = query.Where("children.music_folder_id = ?", folderID)
		}
		if err := query.Group("artist_id3.id").Find(&artists).Error; err != nil {
			s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to query artists"))
			return
		}
		for _, artist := range artists {
			if artist.Name == "" {
				continue
			}
			firstChar := strings.ToUpper(artist.Name[:1])
			indexMap[firstChar] = append(indexMap[firstChar], models.Artist{
				ID:   artist.ID,
				Name: artist.Name,
			})
		}
	} else {
		var children []models.Child
		query := db.Where("is_dir = ?", true).Where("parent = ?", "")
		if hasFolderId {
			query = query.Where("music_folder_id = ?", folderID)
		}

		if err := query.Find(&children).Error; err != nil {
			s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to query artists"))
			return
		}

		for _, child := range children {
			name := child.Title
			if name == "" {
				continue
			}
			firstChar := strings.ToUpper(name[:1])
			indexMap[firstChar] = append(indexMap[firstChar], models.Artist{
				ID:   child.ID,
				Name: child.Title,
			})
		}
	}

	var indexes []models.Index
	for char, artists := range indexMap {
		indexes = append(indexes, models.Index{
			Name:   char,
			Artist: artists,
		})
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Indexes = &models.Indexes{
		LastModified: sc.LastScanTime(),
		Index:        indexes,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetMusicDirectory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}
	db := di.MustInvoke[*gorm.DB](r.Context())
	cfg := di.MustInvoke[*config.Config](r.Context())

	if cfg.Subsonic.BrowseMode == "tag" {
		// Try to find as artist first
		var artist models.ArtistID3
		if err := db.Where("id = ?", id).First(&artist).Error; err == nil {
			// It's an artist, return their albums as directories
			var albums []models.AlbumID3
			db.Scopes(models.AlbumWithStats(false)).
				Joins("JOIN album_artists ON album_artists.album_id3_id = album_id3.id").
				Where("album_artists.artist_id3_id = ?", artist.ID).
				Find(&albums)

			children := make([]models.Child, len(albums))
			for i, a := range albums {
				children[i] = models.Child{
					ID:        a.ID,
					Parent:    artist.ID,
					IsDir:     true,
					Title:     a.Name,
					Artist:    a.Artist,
					ArtistID:  a.ArtistID,
					CoverArt:  a.CoverArt,
					Duration:  a.Duration,
					PlayCount: a.PlayCount,
					Starred:   a.Starred,
					Year:      a.Year,
					Genre:     a.Genre,
					Created:   &a.Created,
				}
			}

			resp := models.NewResponse(models.ResponseStatusOK)
			resp.Directory = &models.Directory{
				ID:      artist.ID,
				Name:    artist.Name,
				Starred: artist.Starred,
				Child:   children,
			}
			s.sendResponse(w, r, resp)
			return
		}

		// Try to find as album
		var album models.AlbumID3
		if err := db.Where("id = ?", id).First(&album).Error; err != nil {
			s.sendResponse(w, r, models.NewErrorResponse(70, "Directory not found"))
			return
		}
		// It's an album, return its songs
		var songs []models.Child
		db.Where("album_id = ?", album.ID).Order("disc_number, track").Find(&songs)

		resp := models.NewResponse(models.ResponseStatusOK)
		resp.Directory = &models.Directory{
			ID:      album.ID,
			Parent:  album.ArtistID,
			Name:    album.Name,
			Starred: album.Starred,
			Child:   songs,
		}
		s.sendResponse(w, r, resp)
		return
	}

	// Fallback to folder-based browsing
	var dir models.Child
	if err := db.Where("id = ? AND is_dir = ?", id, true).First(&dir).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Directory not found"))
		return
	}

	var children []models.Child
	db.Where("parent = ?", dir.ID).Find(&children)

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Directory = &models.Directory{
		ID:            dir.ID,
		Parent:        dir.Parent,
		Name:          dir.Title,
		Starred:       dir.Starred,
		UserRating:    dir.UserRating,
		AverageRating: dir.AverageRating,
		PlayCount:     dir.PlayCount,
		Child:         children,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetGenres(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())
	var genres []models.Genre

	// Query genres with counts
	if err := db.Raw(`
		SELECT g.name, 
		       (SELECT COUNT(*) FROM song_genres WHERE genre_name = g.name) as song_count,
		       (SELECT COUNT(*) FROM album_genres WHERE genre_name = g.name) as album_count
		FROM genres g
	`).Scan(&genres).Error; err != nil {
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
	db := di.MustInvoke[*gorm.DB](r.Context())
	var artists []models.ArtistID3

	if err := db.Scopes(models.ArtistWithStats).Find(&artists).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to query artists"))
		return
	}

	// Group by index
	indexMap := make(map[string][]models.ArtistID3)
	for _, artist := range artists {
		if artist.Name == "" {
			continue
		}
		firstChar := strings.ToUpper(artist.Name[:1])
		indexMap[firstChar] = append(indexMap[firstChar], artist)
	}

	var indexes []models.IndexID3
	for char, artists := range indexMap {
		indexes = append(indexes, models.IndexID3{
			Name:   char,
			Artist: artists,
		})
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Artists = &models.ArtistsID3{
		Index: indexes,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetArtist(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	db := di.MustInvoke[*gorm.DB](r.Context())

	var artist models.ArtistID3
	if err := db.Scopes(models.ArtistWithStats).Where("id = ?", id).First(&artist).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Artist not found"))
		return
	}

	var albums []models.AlbumID3
	db.Scopes(models.AlbumWithStats(false)).
		Joins("JOIN album_artists ON album_artists.album_id3_id = album_id3.id").
		Where("album_artists.artist_id3_id = ?", artist.ID).
		Find(&albums)

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Artist = &models.ArtistWithAlbumsID3{
		ArtistID3: artist,
		Album:     albums,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetAlbum(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	db := di.MustInvoke[*gorm.DB](r.Context())

	var album models.AlbumID3
	if err := db.Scopes(models.AlbumWithStats(true)).Where("id = ?", id).First(&album).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Album not found"))
		return
	}

	var songs []models.Child
	db.Where("album_id = ?", id).Find(&songs)

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Album = &models.AlbumWithSongsID3{
		AlbumID3: album,
		Song:     songs,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetSong(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	db := di.MustInvoke[*gorm.DB](r.Context())

	var song models.Child
	if err := db.Where("id = ? AND is_dir = ?", id, false).First(&song).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Song not found"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Song = &song
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
