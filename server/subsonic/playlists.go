package subsonic

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) handleGetPlaylists(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	user, _ := c.Get("user")
	currentUser := user.(models.User)

	targetUsername := c.DefaultQuery("username", currentUser.Username)
	var playlists []models.PlaylistRecord
	query := db.Model(&models.PlaylistRecord{})
	query = query.Where("owner = ?", targetUsername)
	if targetUsername != currentUser.Username {
		query = query.Where("public = ?", true)
	}
	query.Find(&playlists)

	subsonicPlaylists := make([]models.Playlist, 0, len(playlists))
	for _, p := range playlists {
		var songCount int64
		db.Model(&models.PlaylistSong{}).Where("playlist_id = ?", p.ID).Count(&songCount)

		var duration int
		db.Table("playlist_songs").
			Select("SUM(children.duration)").
			Joins("JOIN children ON children.id = playlist_songs.song_id").
			Where("playlist_songs.playlist_id = ?", p.ID).
			Scan(&duration)

		subsonicPlaylists = append(subsonicPlaylists, models.Playlist{
			ID:        strconv.FormatUint(uint64(p.ID), 10),
			Name:      p.Name,
			Comment:   p.Comment,
			Owner:     p.Owner,
			Public:    p.Public,
			SongCount: int(songCount),
			Duration:  duration,
			Created:   p.CreatedAt,
			Changed:   p.UpdatedAt,
		})
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Playlists = &models.Playlists{
		Playlist: subsonicPlaylists,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetPlaylist(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	id, err := getQueryInt[uint64](c, "id")
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(10, err.Error()))
		return
	}

	var p models.PlaylistRecord
	if err := db.Preload("Songs", func(db *gorm.DB) *gorm.DB {
		return db.Order("position ASC")
	}).First(&p, id).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Playlist not found"))
		return
	}

	user, _ := c.Get("user")
	currentUser := user.(models.User)
	if !p.Public && p.Owner != currentUser.Username {
		s.sendResponse(c, models.NewErrorResponse(70, "Playlist not found"))
		return
	}

	var songs []models.Child
	var duration int
	for _, ps := range p.Songs {
		var song models.Child
		if err := db.First(&song, "id = ?", ps.SongID).Error; err == nil {
			songs = append(songs, song)
			duration += song.Duration
		}
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Playlist = &models.PlaylistWithSongs{
		Playlist: models.Playlist{
			ID:        strconv.FormatUint(uint64(p.ID), 10),
			Name:      p.Name,
			Comment:   p.Comment,
			Owner:     p.Owner,
			Public:    p.Public,
			SongCount: len(songs),
			Duration:  duration,
			Created:   p.CreatedAt,
			Changed:   p.UpdatedAt,
		},
		Entry: songs,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleCreatePlaylist(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	name := c.Query("name")
	if name == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "Playlist name not specified"))
		return
	}

	user, _ := c.Get("user")
	currentUser := user.(models.User)

	p := models.PlaylistRecord{
		Name:  name,
		Owner: currentUser.Username,
	}

	if err := db.Create(&p).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to create playlist"))
		return
	}

	songIDs := c.QueryArray("songId")
	if len(songIDs) > 0 {
		songs := make([]models.PlaylistSong, len(songIDs))
		for i, songID := range songIDs {
			songs[i] = models.PlaylistSong{
				PlaylistID: p.ID,
				SongID:     songID,
				Position:   i,
			}
		}
		db.Create(&songs)
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleUpdatePlaylist(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	id, err := getQueryInt[uint](c, "playlistId")
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(10, err.Error()))
		return
	}

	var p models.PlaylistRecord
	if err := db.First(&p, id).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Playlist not found"))
		return
	}

	user, _ := c.Get("user")
	currentUser := user.(models.User)
	if p.Owner != currentUser.Username {
		s.sendResponse(c, models.NewErrorResponse(0, "Permission denied"))
		return
	}

	if name := c.Query("name"); name != "" {
		p.Name = name
	}
	if comment := c.Query("comment"); comment != "" {
		p.Comment = comment
	}
	if public := c.Query("public"); public != "" {
		p.Public = public == "true"
	}

	db.Save(&p)

	// Handle song additions
	songIDsToAdd := c.QueryArray("songIdToAdd")
	if len(songIDsToAdd) > 0 {
		var maxPos int
		db.Model(&models.PlaylistSong{}).Where("playlist_id = ?", p.ID).Select("COALESCE(MAX(position), -1)").Scan(&maxPos)
		songsToAdd := make([]models.PlaylistSong, len(songIDsToAdd))
		for i, songID := range songIDsToAdd {
			songsToAdd[i] = models.PlaylistSong{
				PlaylistID: p.ID,
				SongID:     songID,
				Position:   maxPos + 1 + i,
			}
		}
		db.Create(&songsToAdd)
	}

	// Handle song removals
	indicesToRemove := c.QueryArray("songIndexToRemove")
	if len(indicesToRemove) > 0 {
		var posList []int
		for _, idxStr := range indicesToRemove {
			if idx, err := strconv.Atoi(idxStr); err == nil {
				posList = append(posList, idx)
			}
		}
		if len(posList) > 0 {
			db.Where("playlist_id = ? AND position IN ?", p.ID, posList).Delete(&models.PlaylistSong{})
			// Re-index
			var songs []models.PlaylistSong
			db.Where("playlist_id = ?", p.ID).Order("position ASC").Find(&songs)
			for i, s := range songs {
				db.Model(&s).Update("position", i)
			}
		}
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleDeletePlaylist(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)
	id, err := getQueryInt[uint](c, "id")
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(10, err.Error()))
		return
	}

	var p models.PlaylistRecord
	if err := db.First(&p, id).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(70, "Playlist not found"))
		return
	}

	user, _ := c.Get("user")
	currentUser := user.(models.User)
	if p.Owner != currentUser.Username {
		s.sendResponse(c, models.NewErrorResponse(0, "Permission denied"))
		return
	}

	db.Delete(&p)

	resp := models.NewResponse(models.ResponseStatusOK)
	s.sendResponse(c, resp)
}
