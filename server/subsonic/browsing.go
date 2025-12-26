package subsonic

import (
	"path/filepath"
	"strings"
	"time"

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
	folderID := c.Query("musicFolderId")

	var children []models.Child
	query := db.Where("is_dir = ?", true).Where("parent = ?", "")
	if folderID != "" {
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
		LastModified: time.Now().Unix(),
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
	db.Where("parent = ?", id).Find(&children)

	resp := models.NewResponse(models.ResponseStatusOK)
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
	db.Raw(`
		SELECT g.name, 
		       (SELECT COUNT(*) FROM song_genres WHERE genre_name = g.name) as song_count,
		       (SELECT COUNT(*) FROM album_genres WHERE genre_name = g.name) as album_count
		FROM genres g
	`).Scan(&genres)

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Genres = &models.Genres{
		Genre: genres,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetArtists(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	var artists []models.ArtistID3

	db.Raw(`
		SELECT a.*, 
		       (SELECT COUNT(*) FROM album_artists WHERE artist_id3_id = a.id) as album_count
		FROM artist_id3 a
	`).Scan(&artists)

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
