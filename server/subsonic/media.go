package subsonic

import (
	"fmt"
	"hash/adler32"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filepath.Base(song.Path)}))
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
		if coverPath := s.findCoverArt(filepath.Dir(song.Path)); coverPath != "" {
			c.File(coverPath)
			return
		}
	}

	// Try to find as album
	var album models.AlbumID3
	if err := db.Where("id = ?", id).First(&album).Error; err == nil {
		// For albums, we need to find one song in the album to get the directory
		var firstSong models.Child
		if err := db.Where("album_id = ?", album.ID).First(&firstSong).Error; err == nil {
			if coverPath := s.findCoverArt(filepath.Dir(firstSong.Path)); coverPath != "" {
				c.File(coverPath)
				return
			}
		}
	}

	// Fallback to a default cover or 404
	c.Status(http.StatusNotFound)
}

func (s *Subsonic) findCoverArt(dir string) string {
	covers := []string{"cover.jpg", "cover.png", "folder.jpg", "folder.png", "front.jpg", "front.png"}
	for _, cover := range covers {
		coverPath := filepath.Join(dir, cover)
		if _, err := os.Stat(coverPath); err == nil {
			return coverPath
		}
	}
	return ""
}

func updateNowPlaying(c *gin.Context, s *Subsonic, id string) {
	user, _ := c.Get("user")
	u := user.(models.User)
	clientName := c.DefaultQuery("c", "Unknown")
	playerId := int(adler32.Checksum([]byte(clientName)))

	// Update in-memory now playing record
	key := fmt.Sprintf("%s:%s", u.Username, clientName)
	s.nowPlaying.Store(key, models.NowPlayingRecord{
		Username:   u.Username,
		ChildID:    id,
		PlayerID:   playerId,
		PlayerName: clientName,
		UpdatedAt:  time.Now(),
	})

	s.sendResponse(c, models.NewResponse(models.ResponseStatusOK))
}

func (s *Subsonic) handleScrobble(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "ID is required"))
		return
	}

	submission := c.Query("submission")
	if submission == "false" {
		// If submission is false, it's just an update now playing call
		updateNowPlaying(c, s, id)
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)
	if err := db.Model(&models.Child{}).Where("id = ?", id).UpdateColumn("play_count", gorm.Expr("play_count + 1")).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to update play count"))
		return
	}

	// Remove now playing record since it's now scrobbled (finished)
	user, _ := c.Get("user")
	u := user.(models.User)
	clientName := c.DefaultQuery("c", "Unknown")
	key := fmt.Sprintf("%s:%s", u.Username, clientName)
	s.nowPlaying.Delete(key)

	s.sendResponse(c, models.NewResponse(models.ResponseStatusOK))
}
