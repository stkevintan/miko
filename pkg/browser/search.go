package browser

import (
	"github.com/stkevintan/miko/models"
)

type SearchOptions struct {
	Query         string
	ArtistCount   int
	ArtistOffset  int
	AlbumCount    int
	AlbumOffset   int
	SongCount     int
	SongOffset    int
	MusicFolderID uint
	HasFolderID   bool
}

func (b *Browser) Search(opts SearchOptions) ([]models.ArtistID3, []models.AlbumID3, []models.Child, error) {
	var artists []models.ArtistID3
	var albums []models.AlbumID3
	var songs []models.Child

	searchQuery := "%" + opts.Query + "%"
	if opts.Query == "" {
		searchQuery = "%"
	}

	artistQuery := b.db.Scopes(models.ArtistWithStats).
		Where("name LIKE ?", searchQuery).Limit(opts.ArtistCount).Offset(opts.ArtistOffset)
	albumQuery := b.db.Scopes(models.AlbumWithStats(false)).
		Where("name LIKE ?", searchQuery).Limit(opts.AlbumCount).Offset(opts.AlbumOffset)
	songQuery := b.db.Where("is_dir = false AND (title LIKE ? OR album LIKE ? OR artist LIKE ?)", searchQuery, searchQuery, searchQuery).
		Limit(opts.SongCount).Offset(opts.SongOffset)

	if opts.HasFolderID {
		var folder models.MusicFolder
		if err := b.db.First(&folder, opts.MusicFolderID).Error; err == nil {
			artistQuery = artistQuery.Joins("JOIN song_artists ON song_artists.artist_id3_id = artist_id3.id").
				Joins("JOIN children ON children.id = song_artists.child_id").
				Where("children.music_folder_id = ?", opts.MusicFolderID).
				Group("artist_id3.id")

			albumQuery = albumQuery.Joins("JOIN children ON children.album_id = album_id3.id").
				Where("children.music_folder_id = ?", opts.MusicFolderID).
				Group("album_id3.id")

			songQuery = songQuery.Where("music_folder_id = ?", opts.MusicFolderID)
		}
	}

	artistQuery.Find(&artists)
	albumQuery.Find(&albums)
	songQuery.Find(&songs)

	return artists, albums, songs, nil
}

func (b *Browser) SearchSongs(query string, count, offset int) ([]models.Child, int64, error) {
	var songs []models.Child
	searchQuery := "%" + query + "%"

	var totalHits int64
	b.db.Model(&models.Child{}).Where("title LIKE ? OR album LIKE ? OR artist LIKE ?", searchQuery, searchQuery, searchQuery).Count(&totalHits)

	err := b.db.Where("title LIKE ? OR album LIKE ? OR artist LIKE ?", searchQuery, searchQuery, searchQuery).
		Limit(count).Offset(offset).Find(&songs).Error

	return songs, totalHits, err
}
