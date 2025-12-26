package subsonic

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) handleSearch(c *gin.Context) {
	query := c.Query("query")
	count, _ := strconv.Atoi(c.DefaultQuery("count", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	db := do.MustInvoke[*gorm.DB](s.injector)

	var songs []models.Child
	searchQuery := "%" + query + "%"

	db.Where("title LIKE ? OR album LIKE ? OR artist LIKE ?", searchQuery, searchQuery, searchQuery).
		Limit(count).Offset(offset).Find(&songs)

	s.sendResponse(c, &models.SubsonicResponse{
		Status:  models.ResponseStatusOK,
		Version: "1.16.1",
		SearchResult: &models.SearchResult{
			Offset:    offset,
			TotalHits: len(songs), // This is not accurate for total hits but fine for now
			Match:     songs,
		},
	})
}

func (s *Subsonic) handleSearch2(c *gin.Context) {
	query := c.Query("query")
	artistCount, _ := strconv.Atoi(c.DefaultQuery("artistCount", "20"))
	artistOffset, _ := strconv.Atoi(c.DefaultQuery("artistOffset", "0"))
	albumCount, _ := strconv.Atoi(c.DefaultQuery("albumCount", "20"))
	albumOffset, _ := strconv.Atoi(c.DefaultQuery("albumOffset", "0"))
	songCount, _ := strconv.Atoi(c.DefaultQuery("songCount", "20"))
	songOffset, _ := strconv.Atoi(c.DefaultQuery("songOffset", "0"))

	db := do.MustInvoke[*gorm.DB](s.injector)

	var artists []models.ArtistID3
	var albums []models.AlbumID3
	var songs []models.Child

	searchQuery := "%" + query + "%"

	db.Where("name LIKE ?", searchQuery).Limit(artistCount).Offset(artistOffset).Find(&artists)
	db.Where("name LIKE ?", searchQuery).Limit(albumCount).Offset(albumOffset).Find(&albums)
	db.Where("title LIKE ?", searchQuery).Limit(songCount).Offset(songOffset).Find(&songs)

	// Convert ArtistID3 to Artist for SearchResult2
	searchArtists := make([]models.Artist, len(artists))
	for i, a := range artists {
		searchArtists[i] = models.Artist{
			ID:   a.ID,
			Name: a.Name,
		}
	}

	// Convert AlbumID3 to Child for SearchResult2
	searchAlbums := make([]models.Child, len(albums))
	for i, a := range albums {
		searchAlbums[i] = models.Child{
			ID:       a.ID,
			Title:    a.Name,
			Artist:   a.Artist,
			ArtistID: a.ArtistID,
			CoverArt: a.CoverArt,
			IsDir:    true,
		}
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SearchResult2 = &models.SearchResult2{
		Artist: searchArtists,
		Album:  searchAlbums,
		Song:   songs,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleSearch3(c *gin.Context) {
	query := c.Query("query")
	artistCount, _ := strconv.Atoi(c.DefaultQuery("artistCount", "20"))
	artistOffset, _ := strconv.Atoi(c.DefaultQuery("artistOffset", "0"))
	albumCount, _ := strconv.Atoi(c.DefaultQuery("albumCount", "20"))
	albumOffset, _ := strconv.Atoi(c.DefaultQuery("albumOffset", "0"))
	songCount, _ := strconv.Atoi(c.DefaultQuery("songCount", "20"))
	songOffset, _ := strconv.Atoi(c.DefaultQuery("songOffset", "0"))

	db := do.MustInvoke[*gorm.DB](s.injector)

	var artists []models.ArtistID3
	var albums []models.AlbumID3
	var songs []models.Child

	searchQuery := "%" + query + "%"

	db.Where("name LIKE ?", searchQuery).Limit(artistCount).Offset(artistOffset).Find(&artists)
	db.Where("name LIKE ?", searchQuery).Limit(albumCount).Offset(albumOffset).Find(&albums)
	db.Where("title LIKE ?", searchQuery).Limit(songCount).Offset(songOffset).Find(&songs)

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SearchResult3 = &models.SearchResult3{
		Artist: artists,
		Album:  albums,
		Song:   songs,
	}
	s.sendResponse(c, resp)
}
