package subsonic

import (
	"net/http"
	"strconv"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/browser"
	"github.com/stkevintan/miko/pkg/di"
	"github.com/stkevintan/miko/pkg/log"
	"gorm.io/gorm"
)

func (s *Subsonic) handleGetPlaylists(w http.ResponseWriter, r *http.Request) {
	br := di.MustInvoke[*browser.Browser](r.Context())
	username := string(di.MustInvoke[models.Username](r.Context()))

	query := r.URL.Query()
	targetUsername := query.Get("username")
	if targetUsername == "" {
		targetUsername = username
	}

	subsonicPlaylists, err := br.GetPlaylists(username, targetUsername)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to retrieve playlists"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Playlists = &models.Playlists{
		Playlist: subsonicPlaylists,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetPlaylist(w http.ResponseWriter, r *http.Request) {
	br := di.MustInvoke[*browser.Browser](r.Context())
	id, err := getQueryInt[uint](r, "id")
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(10, err.Error()))
		return
	}

	playlist, err := br.GetPlaylist(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			s.sendResponse(w, r, models.NewErrorResponse(70, "Playlist not found"))
		} else {
			s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to retrieve playlist"))
		}
		return
	}

	username := string(di.MustInvoke[models.Username](r.Context()))
	if !playlist.Public && playlist.Owner != username {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Playlist not found"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Playlist = playlist
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleCreatePlaylist(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())
	query := r.URL.Query()
	name := query.Get("name")
	if name == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "Playlist name not specified"))
		return
	}

	username := string(di.MustInvoke[models.Username](r.Context()))

	p := models.PlaylistRecord{
		Name:  name,
		Owner: username,
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&p).Error; err != nil {
			return err
		}

		songIDs := query["songId"]
		if len(songIDs) > 0 {
			songs := make([]models.PlaylistSong, len(songIDs))
			for i, songID := range songIDs {
				songs[i] = models.PlaylistSong{
					PlaylistID: p.ID,
					SongID:     songID,
					Position:   i,
				}
			}
			if err := tx.Create(&songs).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to create playlist"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleUpdatePlaylist(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())
	id, err := getQueryInt[uint](r, "playlistId")
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(10, err.Error()))
		return
	}

	var p models.PlaylistRecord
	if err := db.First(&p, id).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Playlist not found"))
		return
	}

	username := string(di.MustInvoke[models.Username](r.Context()))

	if p.Owner != username {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Permission denied"))
		return
	}

	query := r.URL.Query()
	err = db.Transaction(func(tx *gorm.DB) error {
		if name := query.Get("name"); name != "" {
			p.Name = name
		}
		if comment := query.Get("comment"); comment != "" {
			p.Comment = comment
		}
		if public := query.Get("public"); public != "" {
			p.Public = public == "true"
		}

		if err := tx.Save(&p).Error; err != nil {
			return err
		}

		// Handle song additions
		songIDsToAdd := query["songIdToAdd"]
		if len(songIDsToAdd) > 0 {
			var maxPos int
			if err := tx.Model(&models.PlaylistSong{}).Where("playlist_id = ?", p.ID).Select("COALESCE(MAX(position), -1)").Scan(&maxPos).Error; err != nil {
				return err
			}
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
		indicesToRemove := query["songIndexToRemove"]
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
				// Re-index using a single UPDATE with a CTE
				if err := tx.Exec(`
					WITH new_positions AS (
						SELECT id, (ROW_NUMBER() OVER (ORDER BY position)) - 1 AS new_pos
						FROM playlist_songs
						WHERE playlist_id = ?
					)
					UPDATE playlist_songs
					SET position = new_positions.new_pos
					FROM new_positions
					WHERE playlist_songs.id = new_positions.id
				`, p.ID).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to update playlist: "+err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleDeletePlaylist(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())
	id, err := getQueryInt[uint](r, "id")
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(10, err.Error()))
		return
	}

	var p models.PlaylistRecord
	if err := db.First(&p, id).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(70, "Playlist not found"))
		return
	}

	username := string(di.MustInvoke[models.Username](r.Context()))
	if p.Owner != username {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Permission denied"))
		return
	}

	if err := db.Delete(&p).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to delete playlist"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	s.sendResponse(w, r, resp)
}
