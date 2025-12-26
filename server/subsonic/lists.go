package subsonic

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) handleGetAlbumList2(c *gin.Context) {
	listType := c.DefaultQuery("type", "newest")
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	genre := c.Query("genre")
	fromYear, _ := strconv.Atoi(c.DefaultQuery("fromYear", "0"))
	toYear, _ := strconv.Atoi(c.DefaultQuery("toYear", "3000"))

	db := do.MustInvoke[*gorm.DB](s.injector)
	var albums []models.AlbumID3

	query := db.Limit(size).Offset(offset)

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

	if err := query.Find(&albums).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to fetch albums"))
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

	listType := c.DefaultQuery("type", "newest")
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	db := do.MustInvoke[*gorm.DB](s.injector)
	var albums []models.AlbumID3

	query := db.Limit(size).Offset(offset)

	switch listType {
	case "random":
		query = query.Order("RANDOM()")
	case "newest":
		query = query.Order("created DESC")
	case "alphabeticalByName":
		query = query.Order("name ASC")
	case "alphabeticalByArtist":
		query = query.Order("artist ASC")
	default:
		query = query.Order("created DESC")
	}

	if err := query.Find(&albums).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to fetch albums"))
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
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	genre := c.Query("genre")
	fromYear, _ := strconv.Atoi(c.DefaultQuery("fromYear", "0"))
	toYear, _ := strconv.Atoi(c.DefaultQuery("toYear", "3000"))

	db := do.MustInvoke[*gorm.DB](s.injector)
	var songs []models.Child

	query := db.Where("is_dir = ?", false).Limit(size).Order("RANDOM()")

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
