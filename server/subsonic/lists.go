package subsonic

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) getAlbums(c *gin.Context) ([]models.AlbumID3, error) {
	listType := c.DefaultQuery("type", "newest")
	var err error
	size := s.getQueryIntOrDefault(c, "size", 10, &err)
	offset := s.getQueryIntOrDefault(c, "offset", 0, &err)
	genre := c.Query("genre")
	fromYear := s.getQueryIntOrDefault(c, "fromYear", 0, &err)
	toYear := s.getQueryIntOrDefault(c, "toYear", 3000, &err)
	if err != nil {
		return nil, err
	}
	musicFolderId := c.Query("musicFolderId")

	db := do.MustInvoke[*gorm.DB](s.injector)
	var albums []models.AlbumID3

	query := db.Limit(size).Offset(offset)

	if musicFolderId != "" {
		query = query.Joins("JOIN children ON children.album_id = album_id3s.id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("album_id3s.id")
	}

	switch listType {
	case "random":
		query = query.Order("RANDOM()")
	case "newest":
		query = query.Order("created DESC")
	case "alphabeticalByName":
		query = query.Order("name ASC")
	case "alphabeticalByArtist":
		query = query.Order("artist ASC")
	case "byYear":
		query = query.Where("year >= ? AND year <= ?", fromYear, toYear).Order("year DESC")
	case "byGenre":
		if genre != "" {
			query = query.Joins("JOIN album_genres ON album_genres.album_id3_id = album_id3s.id").
				Where("album_genres.genre_name = ?", genre)
		}
	default:
		query = query.Order("created DESC")
	}

	err = query.Find(&albums).Error
	return albums, err
}

func (s *Subsonic) handleGetAlbumList2(c *gin.Context) {
	albums, err := s.getAlbums(c)
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

	albums, err := s.getAlbums(c)
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
	var err error
	size := s.getQueryIntOrDefault(c, "size", 10, &err)
	genre := c.Query("genre")
	fromYear := s.getQueryIntOrDefault(c, "fromYear", 0, &err)
	toYear := s.getQueryIntOrDefault(c, "toYear", 3000, &err)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}
	musicFolderId := c.Query("musicFolderId")

	db := do.MustInvoke[*gorm.DB](s.injector)
	var songs []models.Child

	query := db.Where("is_dir = ?", false).Limit(size).Order("RANDOM()")

	if musicFolderId != "" {
		query = query.Where("music_folder_id = ?", musicFolderId)
	}

	if genre != "" {
		query = query.Joins("JOIN song_genres ON song_genres.child_id = children.id").
			Where("song_genres.genre_name = ?", genre)
	}
	if fromYear > 0 {
		query = query.Where("year >= ?", fromYear)
	}
	if toYear < 3000 {
		query = query.Where("year <= ?", toYear)
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

	var err error
	count := s.getQueryIntOrDefault(c, "count", 10, &err)
	offset := s.getQueryIntOrDefault(c, "offset", 0, &err)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)
	var songs []models.Child
	err = db.Joins("JOIN song_genres ON song_genres.child_id = children.id").
		Where("song_genres.genre_name = ?", genre).
		Limit(count).Offset(offset).
		Find(&songs).Error

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
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.NowPlaying = &models.NowPlaying{
		Entry: []models.NowPlayingEntry{},
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetStarred(c *gin.Context) {
	db := do.MustInvoke[*gorm.DB](s.injector)

	var artists []models.ArtistID3
	db.Where("starred IS NOT NULL").Find(&artists)

	var albums []models.AlbumID3
	db.Where("starred IS NOT NULL").Find(&albums)

	var songs []models.Child
	db.Where("is_dir = ? AND starred IS NOT NULL", false).Find(&songs)

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
	db := do.MustInvoke[*gorm.DB](s.injector)

	var artists []models.ArtistID3
	db.Where("starred IS NOT NULL").Find(&artists)

	var albums []models.AlbumID3
	db.Where("starred IS NOT NULL").Find(&albums)

	var songs []models.Child
	db.Where("is_dir = ? AND starred IS NOT NULL", false).Find(&songs)

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Starred2 = &models.Starred2{
		Artist: artists,
		Album:  albums,
		Song:   songs,
	}
	s.sendResponse(c, resp)
}
