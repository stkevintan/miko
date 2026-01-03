package browser

import (
	"sort"
	"strings"

	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (b *Browser) GetIndexes(mode string, folderID uint, hasFolderId bool, ignoredArticles string) ([]models.Index, error) {
	if mode == "tag" {
		return b.getIndexesByTag(folderID, hasFolderId, ignoredArticles)
	}
	return b.getIndexesByFile(folderID, hasFolderId, ignoredArticles)
}

func (b *Browser) getIndexesByTag(folderID uint, hasFolderId bool, ignoredArticles string) ([]models.Index, error) {
	indexMap := make(map[string][]models.Artist)
	var artists []models.ArtistID3
	query := b.db.Model(&models.ArtistID3{}).Select("artist_id3.id, artist_id3.name")
	query = query.Joins("JOIN song_artists ON song_artists.artist_id3_id = artist_id3.id").
		Joins("JOIN children ON children.id = song_artists.child_id")
	if hasFolderId {
		query = query.Where("children.music_folder_id = ?", folderID)
	}
	if err := query.Group("artist_id3.id").Find(&artists).Error; err != nil {
		return nil, err
	}

	articles := strings.Fields(ignoredArticles)

	for _, artist := range artists {
		if artist.Name == "" {
			continue
		}

		name := stripArticles(artist.Name, articles)

		firstChar := strings.ToUpper(string([]rune(name)[0]))
		indexMap[firstChar] = append(indexMap[firstChar], models.Artist{
			ID:   artist.ID,
			Name: artist.Name,
		})
	}

	return b.mapToIndexes(indexMap), nil
}

func (b *Browser) getIndexesByFile(folderID uint, hasFolderId bool, ignoredArticles string) ([]models.Index, error) {
	indexMap := make(map[string][]models.Artist)
	var children []models.Child
	query := b.db.Model(&models.Child{}).Select("id, title, is_dir, parent").Where("is_dir = ?", true).Where("parent = ?", "")
	if hasFolderId {
		query = query.Where("music_folder_id = ?", folderID)
	}

	if err := query.Find(&children).Error; err != nil {
		return nil, err
	}

	articles := strings.Fields(ignoredArticles)

	for _, child := range children {
		name := child.Title
		if name == "" {
			continue
		}

		sortName := stripArticles(name, articles)

		firstChar := strings.ToUpper(string([]rune(sortName)[0]))
		indexMap[firstChar] = append(indexMap[firstChar], models.Artist{
			ID:   child.ID,
			Name: child.Title,
		})
	}

	return b.mapToIndexes(indexMap), nil
}

func (b *Browser) mapToIndexes(indexMap map[string][]models.Artist) []models.Index {
	var indexes []models.Index
	for char, artists := range indexMap {
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].Name < artists[j].Name
		})
		indexes = append(indexes, models.Index{
			Name:   char,
			Artist: artists,
		})
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].Name < indexes[j].Name
	})
	return indexes
}

func (b *Browser) GetDirectory(mode string, id string) (*models.Directory, error) {
	if mode == "tag" {
		return b.getDirectoryByTag(id)
	}
	return b.getDirectoryByFile(id)
}

func (b *Browser) getDirectoryByTag(id string) (*models.Directory, error) {
	// Try to find as artist first
	var artist models.ArtistID3
	if err := b.db.Model(&models.ArtistID3{}).Select("id, name, starred").Where("id = ?", id).First(&artist).Error; err == nil {
		// It's an artist, return their albums as directories
		var albums []models.AlbumID3
		b.db.Scopes(models.AlbumWithStats(false)).
			Joins("JOIN album_artists ON album_artists.album_id3_id = album_id3.id").
			Where("album_artists.artist_id3_id = ?", artist.ID).
			Order("album_id3.year DESC, album_id3.name ASC").
			Find(&albums)

		children := make([]models.Child, len(albums))
		for i, a := range albums {
			children[i] = models.Child{
				ID:        a.ID,
				Parent:    artist.ID,
				IsDir:     true,
				Title:     a.Name,
				Artist:    a.Artist,
				ArtistID:  a.ArtistID,
				CoverArt:  a.CoverArt,
				Duration:  a.Duration,
				PlayCount: a.PlayCount,
				Starred:   a.Starred,
				Year:      a.Year,
				Genre:     a.Genre,
				Created:   &a.Created,
			}
		}

		return &models.Directory{
			ID:      artist.ID,
			Name:    artist.Name,
			Starred: artist.Starred,
			Child:   children,
		}, nil
	}

	// Try to find as album
	var album models.AlbumID3
	if err := b.db.Model(&models.AlbumID3{}).Select("id, name, starred, artist_id").Where("id = ?", id).First(&album).Error; err == nil {
		// It's an album, return its songs
		var songs []models.Child
		b.db.Where("album_id = ?", album.ID).Order("disc_number, track").Find(&songs)

		return &models.Directory{
			ID:      album.ID,
			Parent:  album.ArtistID,
			Name:    album.Name,
			Starred: album.Starred,
			Child:   songs,
		}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (b *Browser) getDirectoryByFile(id string) (*models.Directory, error) {
	var dir models.Child
	if err := b.db.Where("id = ? AND is_dir = ?", id, true).First(&dir).Error; err != nil {
		return nil, err
	}

	var children []models.Child
	b.db.Where("parent = ?", dir.ID).
		Order("is_dir DESC, title ASC").
		Find(&children)

	return &models.Directory{
		ID:            dir.ID,
		Parent:        dir.Parent,
		Name:          dir.Title,
		Starred:       dir.Starred,
		UserRating:    dir.UserRating,
		AverageRating: dir.AverageRating,
		PlayCount:     dir.PlayCount,
		Child:         children,
	}, nil
}

func (b *Browser) GetGenres() ([]models.Genre, error) {
	var genres []models.Genre
	if err := b.db.Raw(`
		SELECT g.name, 
		       (SELECT COUNT(*) FROM song_genres WHERE genre_name = g.name) as song_count,
		       (SELECT COUNT(*) FROM album_genres WHERE genre_name = g.name) as album_count
		FROM genres g
	`).Scan(&genres).Error; err != nil {
		return nil, err
	}
	return genres, nil
}

func (b *Browser) GetArtists(ignoredArticles string) ([]models.IndexID3, error) {
	var artists []models.ArtistID3
	if err := b.db.Scopes(models.ArtistWithStats).Find(&artists).Error; err != nil {
		return nil, err
	}

	articles := strings.Fields(ignoredArticles)
	indexMap := make(map[string][]models.ArtistID3)
	for _, artist := range artists {
		if artist.Name == "" {
			continue
		}

		name := stripArticles(artist.Name, articles)

		firstChar := strings.ToUpper(string([]rune(name)[0]))
		indexMap[firstChar] = append(indexMap[firstChar], artist)
	}

	var indexes []models.IndexID3
	for char, artists := range indexMap {
		sort.Slice(artists, func(i, j int) bool {
			return artists[i].Name < artists[j].Name
		})
		indexes = append(indexes, models.IndexID3{
			Name:   char,
			Artist: artists,
		})
	}
	sort.Slice(indexes, func(i, j int) bool {
		return indexes[i].Name < indexes[j].Name
	})
	return indexes, nil
}

func (b *Browser) GetArtist(id string) (*models.ArtistWithAlbumsID3, error) {
	var artist models.ArtistID3
	if err := b.db.Scopes(models.ArtistWithStats).Where("id = ?", id).First(&artist).Error; err != nil {
		return nil, err
	}

	var albums []models.AlbumID3
	b.db.Scopes(models.AlbumWithStats(false)).
		Joins("JOIN album_artists ON album_artists.album_id3_id = album_id3.id").
		Where("album_artists.artist_id3_id = ?", artist.ID).
		Order("album_id3.year DESC, album_id3.name ASC").
		Find(&albums)

	return &models.ArtistWithAlbumsID3{
		ArtistID3: artist,
		Album:     albums,
	}, nil
}

func (b *Browser) GetAlbum(id string) (*models.AlbumWithSongsID3, error) {
	var album models.AlbumID3
	if err := b.db.Scopes(models.AlbumWithStats(true)).Where("id = ?", id).First(&album).Error; err != nil {
		return nil, err
	}

	var songs []models.Child
	b.db.Where("album_id = ?", id).
		Order("disc_number, track").
		Find(&songs)

	return &models.AlbumWithSongsID3{
		AlbumID3: album,
		Song:     songs,
	}, nil
}

func (b *Browser) GetSong(id string) (*models.Child, error) {
	var song models.Child
	if err := b.db.Where("id = ? AND is_dir = ?", id, false).First(&song).Error; err != nil {
		return nil, err
	}
	return &song, nil
}

func stripArticles(name string, articles []string) string {
	upperName := strings.ToUpper(name)
	for _, article := range articles {
		prefix := strings.ToUpper(article) + " "
		if strings.HasPrefix(upperName, prefix) {
			return name[len(prefix):]
		}
	}
	return name
}
