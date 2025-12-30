package browser

import (
	"github.com/stkevintan/miko/models"
)

type AlbumListOptions struct {
	Type          string
	Size          int
	Offset        int
	Genre         string
	FromYear      int
	ToYear        int
	MusicFolderID uint
	HasFolderID   bool
}

func (b *Browser) GetAlbums(opts AlbumListOptions) ([]models.AlbumID3, error) {
	var albums []models.AlbumID3
	dbQuery := b.db.Scopes(models.AlbumWithStats(opts.Type == "recent")).Limit(opts.Size).Offset(opts.Offset)

	if opts.HasFolderID {
		dbQuery = dbQuery.Joins("JOIN children ON children.album_id = album_id3.id").
			Where("children.music_folder_id = ?", opts.MusicFolderID).
			Group("album_id3.id")
	}

	switch opts.Type {
	case "random":
		dbQuery = dbQuery.Order("RANDOM()")
	case "newest":
		dbQuery = dbQuery.Order("album_id3.created DESC")
	case "frequent":
		dbQuery = dbQuery.Order("play_count DESC")
	case "recent":
		dbQuery = dbQuery.Where("last_played IS NOT NULL").
			Order("last_played DESC")
	case "starred":
		dbQuery = dbQuery.Where("album_id3.starred IS NOT NULL").Order("album_id3.starred DESC")
	case "alphabeticalByName":
		dbQuery = dbQuery.Order("album_id3.name ASC")
	case "alphabeticalByArtist":
		dbQuery = dbQuery.Order("album_id3.artist ASC")
	case "byYear":
		dbQuery = dbQuery.Where("album_id3.year >= ? AND album_id3.year <= ?", opts.FromYear, opts.ToYear).Order("album_id3.year DESC")
	case "byGenre":
		if opts.Genre != "" {
			dbQuery = dbQuery.Joins("JOIN album_genres ON album_genres.album_id3_id = album_id3.id").
				Where("album_genres.genre_name = ?", opts.Genre)
		}
	default:
		dbQuery = dbQuery.Order("album_id3.created DESC")
	}

	err := dbQuery.Find(&albums).Error
	return albums, err
}

func (b *Browser) GetRandomSongs(opts AlbumListOptions) ([]models.Child, error) {
	var songs []models.Child
	dbQuery := b.db.Where("is_dir = ?", false).Limit(opts.Size).Order("RANDOM()")

	if opts.HasFolderID {
		dbQuery = dbQuery.Where("music_folder_id = ?", opts.MusicFolderID)
	}

	if opts.Genre != "" {
		dbQuery = dbQuery.Joins("JOIN song_genres ON song_genres.child_id = children.id").
			Where("song_genres.genre_name = ?", opts.Genre)
	}
	if opts.FromYear > 0 {
		dbQuery = dbQuery.Where("children.year >= ?", opts.FromYear)
	}
	if opts.ToYear < 3000 {
		dbQuery = dbQuery.Where("children.year <= ?", opts.ToYear)
	}

	err := dbQuery.Find(&songs).Error
	return songs, err
}

func (b *Browser) GetSongsByGenre(genre string, count, offset int, folderID uint, hasFolderID bool) ([]models.Child, error) {
	var songs []models.Child
	dbQuery := b.db.Joins("JOIN song_genres ON song_genres.child_id = children.id").
		Where("song_genres.genre_name = ?", genre)

	if hasFolderID {
		dbQuery = dbQuery.Where("children.music_folder_id = ?", folderID)
	}

	err := dbQuery.Limit(count).Offset(offset).Find(&songs).Error
	return songs, err
}

func (b *Browser) GetStarredItems(folderID uint, hasFolderID bool) ([]models.ArtistID3, []models.AlbumID3, []models.Child, error) {
	var artists []models.ArtistID3
	artistQuery := b.db.Scopes(models.ArtistWithStats).Where("artist_id3.starred IS NOT NULL").Order("artist_id3.starred DESC")
	if hasFolderID {
		artistQuery = artistQuery.Joins("JOIN children ON children.artist_id = artist_id3.id").
			Where("children.music_folder_id = ?", folderID).
			Group("artist_id3.id")
	}
	if err := artistQuery.Find(&artists).Error; err != nil {
		return nil, nil, nil, err
	}

	var albums []models.AlbumID3
	albumQuery := b.db.Scopes(models.AlbumWithStats(false)).Where("album_id3.starred IS NOT NULL").Order("album_id3.starred DESC")
	if hasFolderID {
		albumQuery = albumQuery.Joins("JOIN children ON children.album_id = album_id3.id").
			Where("children.music_folder_id = ?", folderID).
			Group("album_id3.id")
	}
	if err := albumQuery.Find(&albums).Error; err != nil {
		return nil, nil, nil, err
	}

	var songs []models.Child
	songQuery := b.db.Where("children.is_dir = ? AND children.starred IS NOT NULL", false).Order("children.starred DESC")
	if hasFolderID {
		songQuery = songQuery.Where("children.music_folder_id = ?", folderID)
	}
	if err := songQuery.Find(&songs).Error; err != nil {
		return nil, nil, nil, err
	}

	return artists, albums, songs, nil
}
