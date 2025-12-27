package subsonic

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
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
	if err := query.Find(&playlists).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to retrieve playlists"))
		return
	}

	subsonicPlaylists := make([]models.Playlist, 0, len(playlists))
	if len(playlists) > 0 {
		playlistIDs := make([]uint, len(playlists))
		for i, p := range playlists {
			playlistIDs[i] = p.ID
		}

		type PlaylistStats struct {
			PlaylistID uint
			SongCount  int
			Duration   int
		}
		var stats []PlaylistStats
		db.Table("playlist_songs").
			Select("playlist_id, COUNT(playlist_songs.id) as song_count, COALESCE(SUM(children.duration), 0) as duration").
			Joins("LEFT JOIN children ON children.id = playlist_songs.song_id").
			Where("playlist_id IN ?", playlistIDs).
			Group("playlist_id").
			Scan(&stats)

		statsMap := make(map[uint]PlaylistStats)
		for _, s := range stats {
			statsMap[s.PlaylistID] = s
		}

		for _, p := range playlists {
			s := statsMap[p.ID]
			subsonicPlaylists = append(subsonicPlaylists, models.Playlist{
				ID:        strconv.FormatUint(uint64(p.ID), 10),
				Name:      p.Name,
				Comment:   p.Comment,
				Owner:     p.Owner,
				Public:    p.Public,
				SongCount: s.SongCount,
				Duration:  s.Duration,
				Created:   p.CreatedAt,
				Changed:   p.UpdatedAt,
			})
		}
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
	if err := db.First(&p, id).Error; err != nil {
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
	if err := db.Model(&models.Child{}).
		Joins("JOIN playlist_songs ON playlist_songs.song_id = children.id").
		Where("playlist_songs.playlist_id = ?", p.ID).
		Order("playlist_songs.position ASC").
		Find(&songs).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to retrieve songs"))
		return
	}

	var duration int
	for _, song := range songs {
		duration += song.Duration
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
		if err := db.Create(&songs).Error; err != nil {
			s.sendResponse(c, models.NewErrorResponse(0, "Failed to add songs to playlist"))
			return
		}
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

	err = db.Transaction(func(tx *gorm.DB) error {
		if name, ok := c.GetQuery("name"); ok {
			p.Name = name
		}
		if comment, ok := c.GetQuery("comment"); ok {
			p.Comment = comment
		}
		if public, ok := c.GetQuery("public"); ok {
			p.Public = public == "true"
		}

		if err := tx.Save(&p).Error; err != nil {
			return err
		}

		// Handle song additions
		songIDsToAdd := c.QueryArray("songIdToAdd")
		if len(songIDsToAdd) > 0 {
			var maxPos int
			tx.Model(&models.PlaylistSong{}).Where("playlist_id = ?", p.ID).Select("COALESCE(MAX(position), -1)").Scan(&maxPos)
			songsToAdd := make([]models.PlaylistSong, len(songIDsToAdd))
			for i, songID := range songIDsToAdd {
				songsToAdd[i] = models.PlaylistSong{
					PlaylistID: p.ID,
					SongID:     songID,
					Position:   maxPos + 1 + i,
				}
			}
			if err := tx.Create(&songsToAdd).Error; err != nil {
				return err
			}
		}

		// Handle song removals
		indicesToRemove := c.QueryArray("songIndexToRemove")
		if len(indicesToRemove) > 0 {
			var posList []int
			for _, idxStr := range indicesToRemove {
				if idx, err := strconv.Atoi(idxStr); err == nil {
					posList = append(posList, idx)
				} else {
					log.Warn("Invalid songIndexToRemove: %s for playlist %d", idxStr, p.ID)
				}
			}

			if len(posList) > 0 {
				if err := tx.Where("playlist_id = ? AND position IN ?", p.ID, posList).Delete(&models.PlaylistSong{}).Error; err != nil {
					return err
				}
				// Re-index using a single UPDATE with ROW_NUMBER()
				if err := tx.Exec(`
					UPDATE playlist_songs
					SET position = (
						SELECT new_pos
						FROM (
							SELECT id, (ROW_NUMBER() OVER (ORDER BY position)) - 1 AS new_pos
							FROM playlist_songs
							WHERE playlist_id = ?
						) AS cte
						WHERE cte.id = playlist_songs.id
					)
					WHERE playlist_id = ?
				`, p.ID, p.ID).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to update playlist: "+err.Error()))
		return
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

	if err := db.Delete(&p).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to delete playlist"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	s.sendResponse(c, resp)
}
