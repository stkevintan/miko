package subsonic

import (
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

func (s *Subsonic) handleSearch(c *gin.Context) {
	query := c.Query("query")
	var err error
	count := getQueryIntOrDefault(c, "count", 20, &err)
	offset := getQueryIntOrDefault(c, "offset", 0, &err)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}

	db := do.MustInvoke[*gorm.DB](s.injector)

	var songs []models.Child
	searchQuery := "%" + query + "%"

	var totalHits int64
	db.Model(&models.Child{}).Where("title LIKE ? OR album LIKE ? OR artist LIKE ?", searchQuery, searchQuery, searchQuery).Count(&totalHits)

	if err := db.Where("title LIKE ? OR album LIKE ? OR artist LIKE ?", searchQuery, searchQuery, searchQuery).
		Limit(count).Offset(offset).Find(&songs).Error; err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, "Failed to search for songs"))
		return
	}

	s.sendResponse(c, &models.SubsonicResponse{
		Status:  models.ResponseStatusOK,
		Version: "1.16.1",
		SearchResult: &models.SearchResult{
			Offset:    offset,
			TotalHits: int(totalHits),
			Match:     songs,
		},
	})
}

func (s *Subsonic) searchCommon(c *gin.Context) ([]models.ArtistID3, []models.AlbumID3, []models.Child, error) {
	query := c.Query("query")
	var err error
	artistCount := getQueryIntOrDefault(c, "artistCount", 20, &err)
	artistOffset := getQueryIntOrDefault(c, "artistOffset", 0, &err)
	albumCount := getQueryIntOrDefault(c, "albumCount", 20, &err)
	albumOffset := getQueryIntOrDefault(c, "albumOffset", 0, &err)
	songCount := getQueryIntOrDefault(c, "songCount", 20, &err)
	songOffset := getQueryIntOrDefault(c, "songOffset", 0, &err)
	if err != nil {
		return nil, nil, nil, err
	}
	musicFolderId := c.Query("musicFolderId")

	db := do.MustInvoke[*gorm.DB](s.injector)

	var artists []models.ArtistID3
	var albums []models.AlbumID3
	var songs []models.Child

	searchQuery := "%" + query + "%"

	artistQuery := db.Where("name LIKE ?", searchQuery).Limit(artistCount).Offset(artistOffset)
	albumQuery := db.Where("name LIKE ?", searchQuery).Limit(albumCount).Offset(albumOffset)
	songQuery := db.Where("title LIKE ?", searchQuery).Limit(songCount).Offset(songOffset)

	if musicFolderId != "" {
		// For artists and albums, we filter by checking if they have songs in the folder
		artistQuery = artistQuery.Joins("JOIN song_artists ON song_artists.artist_id3_id = artist_id3s.id").
			Joins("JOIN children ON children.id = song_artists.child_id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("artist_id3s.id")

		albumQuery = albumQuery.Joins("JOIN children ON children.album_id = album_id3s.id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("album_id3s.id")

		songQuery = songQuery.Where("music_folder_id = ?", musicFolderId)
	}

	artistQuery.Find(&artists)
	albumQuery.Find(&albums)
	songQuery.Find(&songs)

	return artists, albums, songs, nil
}

func (s *Subsonic) handleSearch2(c *gin.Context) {
	artists, albums, songs, err := s.searchCommon(c)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}

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
	artists, albums, songs, err := s.searchCommon(c)
	if err != nil {
		s.sendResponse(c, models.NewErrorResponse(0, err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SearchResult3 = &models.SearchResult3{
		Artist: artists,
		Album:  albums,
		Song:   songs,
	}
	s.sendResponse(c, resp)
}
