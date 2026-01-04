package browser

import (
	"sort"
	"strings"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
)

func (b *Browser) GetIndexes(folderID uint, hasFolderId bool, ignoredArticles string) ([]models.Index, error) {
	indexMap := make(map[string][]models.Artist)
	var children []models.Child
	query := b.db.Model(&models.Child{}).Select("id, title, is_dir, parent, music_folder_id").Where("is_dir = ?", true).Where("parent = ?", "")
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

func (b *Browser) GetDirectory(id string, offset, limit int) (*models.Directory, error) {
	log.Debug("GetDirectory %s %d %d", id, offset, limit)
	var dir models.Child
	if err := b.db.Where("id = ? AND is_dir = ?", id, true).First(&dir).Error; err != nil {
		log.Debug("GetDirectory dir not found: %s", id)
		return nil, err
	}
	log.Debug("GetDirectory found dir: %s (%s)", dir.Title, dir.Path)

	var children []models.Child
	var total int64
	b.db.Model(&models.Child{}).Where("parent = ?", dir.ID).Count(&total)
	log.Debug("GetDirectory total: %d for parent: %s", total, dir.ID)

	query := b.db.Where("parent = ?", dir.ID)
	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Order("is_dir DESC, title ASC").Find(&children).Error; err != nil {
		log.Error("GetDirectory find error: %v", err)
		return nil, err
	}
	log.Debug("GetDirectory found %d children", len(children))

	var parents []models.Child
	if dir.Parent != "" {
		// Use a recursive CTE to fetch all ancestors in a single query.
		// This avoids the N+1 query problem when navigating deep directory structures.
		if err := b.db.Raw(`
			WITH RECURSIVE ancestors AS (
				SELECT * FROM children WHERE id = ?
				UNION ALL
				SELECT c.* FROM children c
				JOIN ancestors a ON c.id = a.parent
			)
			SELECT * FROM ancestors
		`, dir.Parent).Scan(&parents).Error; err != nil {
			log.Error("GetDirectory ancestors error: %v", err)
		}

		// The CTE returns ancestors from leaf to root (closest parent first).
		// Reverse them to get root-to-leaf order for breadcrumbs.
		for i, j := 0, len(parents)-1; i < j; i, j = i+1, j-1 {
			parents[i], parents[j] = parents[j], parents[i]
		}
	}

	return &models.Directory{
		ID:            dir.ID,
		Parent:        dir.Parent,
		Name:          dir.Title,
		Starred:       dir.Starred,
		UserRating:    dir.UserRating,
		AverageRating: dir.AverageRating,
		PlayCount:     dir.PlayCount,
		Child:         children,
		TotalCount:    total,
		Parents:       parents,
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
