package subsonic

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func getAlbums(c *gin.Context, s *Subsonic) ([]models.AlbumID3, error) {
	listType := c.DefaultQuery("type", "newest")
	size := getQueryIntOrDefault(c, "size", 10)
	offset := getQueryIntOrDefault(c, "offset", 0)
	genre := c.Query("genre")
	fromYear := getQueryIntOrDefault(c, "fromYear", 0)
	toYear := getQueryIntOrDefault(c, "toYear", 3000)

	db := do.MustInvoke[*gorm.DB](s.injector)
	var albums []models.AlbumID3

	query := db.Limit(size).Offset(offset)

	musicFolderId, err := getQueryInt[uint](c, "musicFolderId")
	if err == nil {
		query = query.Joins("JOIN children ON children.album_id = album_id3.id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("album_id3.id")
	}

	switch listType {
	case "random":
		query = query.Order("RANDOM()")
	case "newest":
		query = query.Order("album_id3.created DESC")
	case "alphabeticalByName":
		query = query.Order("album_id3.name ASC")
	case "alphabeticalByArtist":
		query = query.Order("album_id3.artist ASC")
	case "byYear":
		query = query.Where("album_id3.year >= ? AND album_id3.year <= ?", fromYear, toYear).Order("album_id3.year DESC")
	case "byGenre":
		if genre != "" {
			query = query.Joins("JOIN album_genres ON album_genres.album_id3_id = album_id3.id").
				Where("album_genres.genre_name = ?", genre)
		}
	default:
		query = query.Order("album_id3.created DESC")
	}

	err = query.Find(&albums).Error
	return albums, err
}

func (s *Subsonic) handleGetAlbumList2(c *gin.Context) {
	albums, err := getAlbums(c, s)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.AlbumList2 = &models.AlbumList2{
		Album: albums,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetAlbumList(c *gin.Context) {
	// getAlbumList is the older version, we can just wrap handleGetAlbumList2
	// but it returns SearchResult instead of AlbumList2 in some versions?
	// Actually it returns <albumList> which is same as <albumList2> but with different element names.
	// For simplicity, let's just use the same logic but return AlbumList.

	albums, err := getAlbums(c, s)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}

	// Convert AlbumID3 to Child for AlbumList (which uses Child elements)
	children := make([]models.Child, len(albums))
	for i, a := range albums {
		children[i] = models.Child{
			ID:       a.ID,
			Title:    a.Name,
			Artist:   a.Artist,
			ArtistID: a.ArtistID,
			CoverArt: a.CoverArt,
			IsDir:    true,
			Created:  &a.Created,
			Year:     a.Year,
			Genre:    a.Genre,
		}
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.AlbumList = &models.AlbumList{
		Album: children,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetRandomSongs(c *gin.Context) {
	size := getQueryIntOrDefault(c, "size", 10)
	genre := c.Query("genre")
	fromYear := getQueryIntOrDefault(c, "fromYear", 0)
	toYear := getQueryIntOrDefault(c, "toYear", 3000)

	db := do.MustInvoke[*gorm.DB](s.injector)
	var songs []models.Child

	query := db.Where("is_dir = ?", false).Limit(size).Order("RANDOM()")

	// Optional musicFolderId filter
	musicFolderId, err := getQueryInt[uint](c, "musicFolderId")
	if err == nil {
		query = query.Where("music_folder_id = ?", musicFolderId)
	}

	if genre != "" {
		query = query.Joins("JOIN song_genres ON song_genres.child_id = children.id").
			Where("song_genres.genre_name = ?", genre)
	}
	if fromYear > 0 {
		query = query.Where("children.year >= ?", fromYear)
	}
	if toYear < 3000 {
		query = query.Where("children.year <= ?", toYear)
	}

	if err := query.Find(&songs).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to fetch songs"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.RandomSongs = &models.Songs{
		Song: songs,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetSongsByGenre(c *gin.Context) {
	genre := c.Query("genre")
	if genre == "" {
		s.sendResponse(c, models.NewErrorResponse(10, "Genre is required"))
		return
	}

	count := getQueryIntOrDefault(c, "count", 10)
	offset := getQueryIntOrDefault(c, "offset", 0)

	db := do.MustInvoke[*gorm.DB](s.injector)
	var songs []models.Child
	query := db.Joins("JOIN song_genres ON song_genres.child_id = children.id").
		Where("song_genres.genre_name = ?", genre)

	musicFolderId, err := getQueryInt[uint](c, "musicFolderId")
	if err == nil {
		query = query.Where("children.music_folder_id = ?", musicFolderId)
	}

	err = query.Limit(count).Offset(offset).Find(&songs).Error

	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SongsByGenre = &models.Songs{
		Song: songs,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetNowPlaying(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)

	tenMinutesAgo := time.Now().Add(-10 * time.Minute)
	entries := make([]models.NowPlayingEntry, 0)

	s.nowPlaying.Range(func(key, value interface{}) bool {
		record, ok := value.(models.NowPlayingRecord)
		if !ok {
			return true
		}

		// Clean up records older than 10 minutes
		if record.UpdatedAt.Before(tenMinutesAgo) {
			s.nowPlaying.Delete(key)
			return true
		}

		var song models.Child
		if err := db.Where("id = ?", record.ChildID).First(&song).Error; err == nil {
			entries = append(entries, models.NowPlayingEntry{
				Child:      song,
				Username:   record.Username,
				MinutesAgo: int(time.Since(record.UpdatedAt).Minutes()),
				PlayerID:   record.PlayerID,
				PlayerName: record.PlayerName,
			})
		}
		return true
	})

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.NowPlaying = &models.NowPlaying{
		Entry: entries,
	}
	s.sendResponse(c, resp)
}

func getStarredItems(c *gin.Context, s *Subsonic) ([]models.ArtistID3, []models.AlbumID3, []models.Child, error) {
	db := do.MustInvoke[*gorm.DB](s.injector)

	var artists []models.ArtistID3
	artistQuery := db.Where("starred IS NOT NULL")
	musicFolderId, err := getQueryInt[uint](c, "musicFolderId")
	musicFolderExists := err == nil
	if musicFolderExists {
		artistQuery = artistQuery.Joins("JOIN children ON children.artist_id = artist_id3.id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("artist_id3.id")
	}
	if err := artistQuery.Find(&artists).Error; err != nil {
		return nil, nil, nil, err
	}

	var albums []models.AlbumID3
	albumQuery := db.Where("starred IS NOT NULL")
	if musicFolderExists {
		albumQuery = albumQuery.Joins("JOIN children ON children.album_id = album_id3.id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("album_id3.id")
	}
	if err := albumQuery.Find(&albums).Error; err != nil {
		return nil, nil, nil, err
	}

	var songs []models.Child
	songQuery := db.Where("is_dir = ? AND starred IS NOT NULL", false)
	if musicFolderExists {
		songQuery = songQuery.Where("music_folder_id = ?", musicFolderId)
	}
	if err := songQuery.Find(&songs).Error; err != nil {
		return nil, nil, nil, err
	}

	return artists, albums, songs, nil
}

func (s *Subsonic) handleGetStarred(c *gin.Context) {
	artists, albums, songs, err := getStarredItems(c, s)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}

	// Convert ArtistID3 to Artist
	starredArtists := make([]models.Artist, len(artists))
	for i, a := range artists {
		starredArtists[i] = models.Artist{
			ID:             a.ID,
			Name:           a.Name,
			ArtistImageUrl: a.ArtistImageUrl,
			Starred:        a.Starred,
		}
	}

	// Convert AlbumID3 to Child
	starredAlbums := make([]models.Child, len(albums))
	for i, a := range albums {
		starredAlbums[i] = models.Child{
			ID:       a.ID,
			Title:    a.Name,
			Artist:   a.Artist,
			ArtistID: a.ArtistID,
			CoverArt: a.CoverArt,
			IsDir:    true,
			Created:  &a.Created,
			Year:     a.Year,
			Genre:    a.Genre,
			Starred:  a.Starred,
		}
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Starred = &models.Starred{
		Artist: starredArtists,
		Album:  starredAlbums,
		Song:   songs,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetStarred2(c *gin.Context) {
	artists, albums, songs, err := getStarredItems(c, s)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Starred2 = &models.Starred2{
		Artist: artists,
		Album:  albums,
		Song:   songs,
	}
	s.sendResponse(c, resp)
}
