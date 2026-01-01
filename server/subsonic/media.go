package subsonic

import (
	"fmt"
	"hash/adler32"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/scanner"
	"github.com/stkevintan/miko/pkg/tags"
	"gorm.io/gorm"
)

var lrcRegex = regexp.MustCompile(`^\[(\d+):(\d+)\.(\d+)\](.*)$`)

func (s *Subsonic) handleStream(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var song models.Child
	if err := db.Model(&models.Child{}).Select("path, is_dir").Where("id = ?", id).First(&song).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Song not found"))
		return
	}

	if song.IsDir {
		s.sendResponse(w, r, models.NewErrorResponse(70, "ID is a directory"))
		return
	}

	if _, err := os.Stat(song.Path); err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "File not found on disk"))
		return
	}

	log.Debug("Streaming file: %s", song.Path)
	safeServeFile(w, r, song.Path)
}

func (s *Subsonic) handleDownload(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var song models.Child
	if err := db.Model(&models.Child{}).Select("path").Where("id = ?", id).First(&song).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Song not found"))
		return
	}

	if _, err := os.Stat(song.Path); err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "File not found on disk"))
		return
	}

	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filepath.Base(song.Path)}))
	safeServeFile(w, r, song.Path)
}

func (s *Subsonic) handleGetCoverArt(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	cfg := di.MustInvoke[*config.Config](r.Context())
	cacheDir := scanner.GetCoverCacheDir(cfg)

	// Try to serve from cache first
	cachePath := filepath.Join(cacheDir, id)
	if data, err := os.ReadFile(cachePath); err == nil && len(data) > 0 {
		contentType := http.DetectContentType(data)
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())

	var path string
	if strings.HasPrefix(id, "al-") {
		realID := id[3:]
		var album models.AlbumID3
		if err := db.Model(&models.AlbumID3{}).Select("id").Where("id = ?", realID).First(&album).Error; err == nil {
			var firstSong models.Child
			if err := db.Model(&models.Child{}).Select("path").Where("album_id = ?", album.ID).First(&firstSong).Error; err == nil {
				path = firstSong.Path
			}
		}
	} else if strings.HasPrefix(id, "ar-") {
		realID := id[3:]
		var artist models.ArtistID3
		if err := db.Model(&models.ArtistID3{}).Select("id").Where("id = ?", realID).First(&artist).Error; err == nil {
			var firstSong models.Child
			if err := db.Table("children").
				Select("children.id, children.path").
				Joins("JOIN song_artists ON song_artists.child_id = children.id").
				Where("song_artists.artist_id3_id = ?", artist.ID).
				First(&firstSong).Error; err == nil {
				path = firstSong.Path
			}
		}
	} else {
		var song models.Child
		if err := db.Select("id,path").Where("id = ?", id).First(&song).Error; err == nil {
			path = song.Path
		}
	}

	if path != "" {
		if data, err := tags.ReadImage(path); err == nil && len(data) > 0 {
			// Cache it for next time
			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				log.Warn("Failed to create cover art cache directory %q: %v", cacheDir, err)
			} else if err := os.WriteFile(filepath.Join(cacheDir, id), data, 0644); err != nil {
				log.Warn("Failed to write cover art to cache for id %s: %v", id, err)
			}

			contentType := http.DetectContentType(data)
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}
	}

	// Fallback to a default cover or 404
	w.WriteHeader(http.StatusNotFound)
}

func (s *Subsonic) handleGetLyrics(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	artist := query.Get("artist")
	title := query.Get("title")

	if artist == "" || title == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "Artist and title are required"))
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var song models.Child
	if err := db.Model(&models.Child{}).Select("artist, title, lyrics").Where("artist = ? AND title = ?", artist, title).First(&song).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Lyrics not found"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Lyrics = &models.Lyrics{
		Artist: song.Artist,
		Title:  song.Title,
		Value:  song.Lyrics,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetLyricsBySongId(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	var song models.Child
	if err := db.Model(&models.Child{}).Select("lyrics, artist, title").Where("id = ?", id).First(&song).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Lyrics not found"))
		return
	}

	if song.Lyrics == "" {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Lyrics not found"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)

	rows := strings.Split(song.Lyrics, "\n")
	lines := make([]models.LyricsLine, 0, len(rows))
	synced := true
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		matches := lrcRegex.FindStringSubmatch(row)
		if len(matches) == 5 {
			min, _ := strconv.Atoi(matches[1])
			sec, _ := strconv.Atoi(matches[2])
			msStr := matches[3]
			ms, _ := strconv.Atoi(msStr)
			if len(msStr) == 2 {
				ms *= 10
			}
			text := strings.TrimSpace(matches[4])

			startTime := (min*60+sec)*1000 + ms
			lines = append(lines, models.LyricsLine{
				Start: startTime,
				Value: text,
			})
		} else {
			// If any line is non-synced, mark whole lyrics as non-synced
			synced = false
			// Non-synced line
			lines = append(lines, models.LyricsLine{
				Value: row,
			})
		}
	}

	resp.LyricsList = &models.LyricsList{
		StructuredLyrics: []models.StructuredLyrics{
			{
				Synced:        synced,
				Lang:          "xxx",
				DisplayArtist: song.Artist,
				DisplayTitle:  song.Title,
				Lines:         lines,
			},
		},
	}
	log.Debug("Returning lyrics for song ID %s: %+v", id, resp.LyricsList)
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetAvatar(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "Username is required"))
		return
	}

	cfg := di.MustInvoke[*config.Config](r.Context())
	avatarDir := filepath.Join(cfg.Subsonic.DataDir, "avatars")

	filename := scanner.GenerateHash(username)

	extensions := []string{".jpg", ".png"}
	for _, ext := range extensions {
		avatarPath := filepath.Join(avatarDir, filename+ext)
		if _, err := os.Stat(avatarPath); err == nil {
			http.ServeFile(w, r, avatarPath)
			return
		} else if !os.IsNotExist(err) {
			log.Error("Error accessing avatar %s: %v", avatarPath, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func updateNowPlaying(w http.ResponseWriter, r *http.Request, s *Subsonic, id string) {
	username := string(di.MustInvoke[models.Username](r.Context()))
	clientName := r.URL.Query().Get("c")
	if clientName == "" {
		clientName = "Unknown"
	}
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

	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}

func (s *Subsonic) handleScrobble(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	id := query.Get("id")
	if id == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "ID is required"))
		return
	}

	submission := query.Get("submission")
	if submission == "false" {
		// If submission is false, it's just an update now playing call
		updateNowPlaying(w, r, s, id)
		return
	}

	db := di.MustInvoke[*gorm.DB](r.Context())
	if err := db.Model(&models.Child{}).Where("id = ?", id).Updates(map[string]interface{}{
		"play_count":  gorm.Expr("play_count + 1"),
		"last_played": time.Now(),
	}).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to update play count"))
		return
	}

	// Remove now playing record since it's now scrobbled (finished)
	username := string(di.MustInvoke[models.Username](r.Context()))
	clientName := query.Get("c")
	if clientName == "" {
		clientName = "Unknown"
	}
	key := fmt.Sprintf("%s:%s", username, clientName)
	s.nowPlaying.Delete(key)

	s.sendResponse(w, r, models.NewResponse(models.ResponseStatusOK))
}
