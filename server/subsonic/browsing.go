package subsonic

import (
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) handleGetMusicFolders(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	cfg := do.MustInvoke[*config.Config](s.injector)

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
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetIndexes(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	folderID, err := getQueryInt[uint](c, "musicFolderId")
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(10, err.Error()))
		return
	}

	var children []models.Child
	query := db.Where("is_dir = ?", true).Where("parent = ?", "")
	if folderID != 0 {
		query = query.Where("music_folder_id = ?", folderID)
	}
	query.Find(&children)

	// Group by first letter
	indexMap := make(map[string][]models.Artist)
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

	var indexes []models.Index
	for char, artists := range indexMap {
		indexes = append(indexes, models.Index{
			Name:   char,
			Artist: artists,
		})
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Indexes = &models.Indexes{
		LastModified: lastScanTime.Load(),
		Index:        indexes,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetMusicDirectory(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}
	db := do.MustInvoke[*gorm.DB](s.injector)
	var dir models.Child
	if err := db.Where("id = ? AND is_dir = ?", id, true).First(&dir).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Directory not found"))
		return
	}

	var children []models.Child
	db.Where("parent = ?", dir.ID).Find(&children)

	resp := models.NewResponse(models.ResponseStatusOK)
	// TODO: nested directories??
	resp.Directory = &models.Directory{
		ID:    dir.ID,
		Name:  dir.Title,
		Child: children,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetGenres(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	var genres []models.Genre

	// Query genres with counts
	if err := db.Raw(`
		SELECT g.name, 
		       (SELECT COUNT(*) FROM song_genres WHERE genre_name = g.name) as song_count,
		       (SELECT COUNT(*) FROM album_genres WHERE genre_name = g.name) as album_count
		FROM genres g
	`).Scan(&genres).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to query genres"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Genres = &models.Genres{
		Genre: genres,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetArtists(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	var artists []models.ArtistID3

	if err := db.Raw(`
		SELECT a.*, 
		       (SELECT COUNT(*) FROM album_artists WHERE artist_id3_id = a.id) as album_count
		FROM artist_id3 a
	`).Scan(&artists).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to query artists"))
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
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetArtist(c *gin.Context) {
	id := c.Query("id")
	db := do.MustInvoke[*gorm.DB](s.injector)

	var artist models.ArtistID3
	if err := db.Where("id = ?", id).First(&artist).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Artist not found"))
		return
	}

	var albums []models.AlbumID3
	db.Model(&artist).Association("Albums").Find(&albums)

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Artist = &models.ArtistWithAlbumsID3{
		ArtistID3: artist,
		Album:     albums,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetAlbum(c *gin.Context) {
	id := c.Query("id")
	db := do.MustInvoke[*gorm.DB](s.injector)

	var album models.AlbumID3
	if err := db.Where("id = ?", id).First(&album).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Album not found"))
		return
	}

	var songs []models.Child
	db.Where("album_id = ?", id).Find(&songs)

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Album = &models.AlbumWithSongsID3{
		AlbumID3: album,
		Song:     songs,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetSong(c *gin.Context) {
	id := c.Query("id")
	db := do.MustInvoke[*gorm.DB](s.injector)

	var song models.Child
	if err := db.Where("id = ? AND is_dir = ?", id, false).First(&song).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Song not found"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Song = &song
	s.sendResponse(c, resp)
}

// TODO: Use music provider to get real data
func (s *Subsonic) handleGetArtistInfo2(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty info
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ArtistInfo2 = &models.ArtistInfo2{}
	s.sendResponse(c, resp)
}

// TODO: Use music provider to get real data
func (s *Subsonic) handleGetAlbumInfo2(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty info
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.AlbumInfo = &models.AlbumInfo{}
	s.sendResponse(c, resp)
}

// TODO: Use music provider to get real data
func (s *Subsonic) handleGetSimilarSongs2(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty list
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SimilarSongs2 = &models.SimilarSongs2{
		Song: []models.Child{},
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetAlbumInfo(c *gin.Context) {
	s.handleGetAlbumInfo2(c)
}

func (s *Subsonic) handleGetArtistInfo(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty info
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ArtistInfo = &models.ArtistInfo{}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetSimilarSongs(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty list
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SimilarSongs = &models.SimilarSongs{
		Song: []models.Child{},
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetTopSongs(c *gin.Context) {
	artist := c.Query("artist")
	if artist == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "Artist is required"))
		return
	}

	// For now, we don't have external metadata provider, so return empty list
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.TopSongs = &models.TopSongs{
		Song: []models.Child{},
	}
	s.sendResponse(c, resp)
}
