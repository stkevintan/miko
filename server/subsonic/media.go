package subsonic

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) handleStream(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)
	var song models.Child
	if err := db.Where("id = ?", id).First(&song).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Song not found"))
		return
	}

	if song.IsDir {
		s.sendResponse(c, models.NewErrorResponse(70, "ID is a directory"))
		return
	}

	c.File(song.Path)
}

func (s *Subsonic) handleDownload(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)
	var song models.Child
	if err := db.Where("id = ?", id).First(&song).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Song not found"))
		return
	}

	c.Header("Content-Disposition", "attachment; filename=\""+filepath.Base(song.Path)+"\"")
	c.File(song.Path)
}

func (s *Subsonic) handleGetCoverArt(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)

	// Try to find as song first
	var song models.Child
	if err := db.Where("id = ?", id).First(&song).Error; err == nil {
		// Look for cover art in the same directory
		dir := filepath.Dir(song.Path)
		covers := []string{"cover.jpg", "cover.png", "folder.jpg", "folder.png", "front.jpg", "front.png"}
		for _, cover := range covers {
			coverPath := filepath.Join(dir, cover)
			if _, err := os.Stat(coverPath); err == nil {
				c.File(coverPath)
				return
			}
		}
	}

	// Try to find as album
	var album models.AlbumID3
	if err := db.Where("id = ?", id).First(&album).Error; err == nil {
		// For albums, we need to find one song in the album to get the directory
		var firstSong models.Child
		if err := db.Where("album_id = ?", album.ID).First(&firstSong).Error; err == nil {
			dir := filepath.Dir(firstSong.Path)
			covers := []string{"cover.jpg", "cover.png", "folder.jpg", "folder.png", "front.jpg", "front.png"}
			for _, cover := range covers {
				coverPath := filepath.Join(dir, cover)
				if _, err := os.Stat(coverPath); err == nil {
					c.File(coverPath)
					return
				}
			}
		}
	}

	// Fallback to a default cover or 404
	c.Status(http.StatusNotFound)
}
