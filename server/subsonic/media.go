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
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"go.senan.xyz/taglib"
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

	cfg := do.MustInvoke[*config.Config](s.injector)
	cacheDir := filepath.Join(cfg.Subsonic.CacheDir, "covers")

	// Try to serve from cache first
	cachePath := filepath.Join(cacheDir, id)
	if data, err := os.ReadFile(cachePath); err == nil && len(data) > 0 {
		contentType := http.DetectContentType(data)
		c.Data(http.StatusOK, contentType, data)
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)

	var path string
	// Try to find as song first
	var song models.Child
	if err := db.Where("id = ?", id).First(&song).Error; err == nil {
		path = song.Path
	} else {
		// Try to find as album
		var album models.AlbumID3
		if err := db.Where("id = ?", id).First(&album).Error; err == nil {
			// For albums, we need to find one song in the album to get the file
			var firstSong models.Child
			if err := db.Where("album_id = ?", album.ID).First(&firstSong).Error; err == nil {
				path = firstSong.Path
			}
		}
	}

	if path != "" {
		if data, err := taglib.ReadImage(path); err == nil && len(data) > 0 {
			// Cache it for next time
			os.MkdirAll(cacheDir, 0755)
			os.WriteFile(filepath.Join(cacheDir, id), data, 0644)

			contentType := http.DetectContentType(data)
			c.Data(http.StatusOK, contentType, data)
			return
		}
	}

	// Fallback to a default cover or 404
	c.Status(http.StatusNotFound)
}

func (s *Subsonic) handleGetLyrics(c *gin.Context) {
	artist := c.Query("artist")
	title := c.Query("title")

	if artist == "" || title == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "Artist and title are required"))
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)
	var song models.Child
	if err := db.Where("artist = ? AND title = ?", artist, title).First(&song).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Lyrics not found"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Lyrics = &models.Lyrics{
		Artist: song.Artist,
		Title:  song.Title,
		Value:  song.Lyrics,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetAvatar(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "Username is required"))
		return
	}

	avatarPath := filepath.Join("data", "avatars", username+".jpg")
	if _, err := os.Stat(avatarPath); err != nil {
		avatarPath = filepath.Join("data", "avatars", username+".png")
		if _, err := os.Stat(avatarPath); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
	}

	c.File(avatarPath)
}

func updateNowPlaying(c *gin.Context, s *Subsonic, id string) {
	username, err := getAuthUsername(c)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(20, "Authentication required"))
		return
	}
	clientName := c.DefaultQuery("c", "Unknown")
	playerId := int(adler32.Checksum([]byte(clientName)))

	// Update in-memory now playing record
	key := fmt.Sprintf("%s:%s", username, clientName)
	s.nowPlaying.Store(key, models.NowPlayingRecord{
		Username:   username,
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
	username, err := getAuthUsername(c)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Internal server error"))
		return
	}
	clientName := c.DefaultQuery("c", "Unknown")
	key := fmt.Sprintf("%s:%s", username, clientName)
	s.nowPlaying.Delete(key)

	s.sendResponse(c, models.NewResponse(models.ResponseStatusOK))
}
