package browser

import (
	"strconv"

	"github.com/stkevintan/miko/models"
)

func (b *Browser) GetPlaylists(username, targetUsername string) ([]models.Playlist, error) {
	var playlists []models.PlaylistRecord
	dbQuery := b.db.Model(&models.PlaylistRecord{})
	dbQuery = dbQuery.Where("owner = ?", targetUsername)
	if targetUsername != username {
		dbQuery = dbQuery.Where("public = ?", true)
	}
	if err := dbQuery.Find(&playlists).Error; err != nil {
		return nil, err
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
		err := b.db.Table("playlist_songs").
			Select("playlist_id, COUNT(playlist_songs.id) as song_count, COALESCE(SUM(children.duration), 0) as duration").
			Joins("LEFT JOIN children ON children.id = playlist_songs.song_id").
			Where("playlist_id IN ?", playlistIDs).
			Group("playlist_id").
			Scan(&stats).Error
		if err != nil {
			return nil, err
		}

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
	return subsonicPlaylists, nil
}

func (b *Browser) GetPlaylist(id uint) (*models.PlaylistWithSongs, error) {
	var p models.PlaylistRecord
	if err := b.db.First(&p, id).Error; err != nil {
		return nil, err
	}

	var songs []models.Child
	err := b.db.Table("children").
		Joins("JOIN playlist_songs ON playlist_songs.song_id = children.id").
		Where("playlist_songs.playlist_id = ?", id).
		Order("playlist_songs.position ASC").
		Find(&songs).Error
	if err != nil {
		return nil, err
	}

	var duration int
	for _, song := range songs {
		duration += song.Duration
	}

	return &models.PlaylistWithSongs{
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
	}, nil
}
