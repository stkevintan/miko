package subsonic

import (
	"net/http"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"gorm.io/gorm"
)

func (s *Subsonic) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	count := getQueryIntOrDefault(r, "count", 20)
	offset := getQueryIntOrDefault(r, "offset", 0)

	db := di.MustInvoke[*gorm.DB](r.Context())

	var songs []models.Child
	searchQuery := "%" + query + "%"

	var totalHits int64
	db.Model(&models.Child{}).Where("title LIKE ? OR album LIKE ? OR artist LIKE ?", searchQuery, searchQuery, searchQuery).Count(&totalHits)

	if err := db.Where("title LIKE ? OR album LIKE ? OR artist LIKE ?", searchQuery, searchQuery, searchQuery).
		Limit(count).Offset(offset).Find(&songs).Error; err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to search for songs"))
		return
	}

	s.sendResponse(w, r, &models.SubsonicResponse{
		Status:  models.ResponseStatusOK,
		Version: "1.16.1",
		SearchResult: &models.SearchResult{
			Offset:    offset,
			TotalHits: int(totalHits),
			Match:     songs,
		},
	})
}

func (s *Subsonic) searchCommon(r *http.Request) ([]models.ArtistID3, []models.AlbumID3, []models.Child, error) {
	query := r.URL.Query().Get("query")

	artistCount := getQueryIntOrDefault(r, "artistCount", 20)
	artistOffset := getQueryIntOrDefault(r, "artistOffset", 0)
	albumCount := getQueryIntOrDefault(r, "albumCount", 20)
	albumOffset := getQueryIntOrDefault(r, "albumOffset", 0)
	songCount := getQueryIntOrDefault(r, "songCount", 20)
	songOffset := getQueryIntOrDefault(r, "songOffset", 0)

	db := di.MustInvoke[*gorm.DB](r.Context())

	var artists []models.ArtistID3
	var albums []models.AlbumID3
	var songs []models.Child

	searchQuery := "%" + query + "%"

	artistQuery := db.Where("name LIKE ?", searchQuery).Limit(artistCount).Offset(artistOffset)
	albumQuery := db.Where("name LIKE ?", searchQuery).Limit(albumCount).Offset(albumOffset)
	songQuery := db.Where("title LIKE ?", searchQuery).Limit(songCount).Offset(songOffset)
	// Optional musicFolderId filter
	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	if err == nil {
		// For artists and albums, we filter by checking if they have songs in the folder
		artistQuery = artistQuery.Joins("JOIN song_artists ON song_artists.artist_id3_id = artist_id3.id").
			Joins("JOIN children ON children.id = song_artists.child_id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("artist_id3.id")

		albumQuery = albumQuery.Joins("JOIN children ON children.album_id = album_id3.id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("album_id3.id")

		songQuery = songQuery.Where("music_folder_id = ?", musicFolderId)
	}

	artistQuery.Find(&artists)
	albumQuery.Find(&albums)
	songQuery.Find(&songs)

	return artists, albums, songs, nil
}

func (s *Subsonic) handleSearch2(w http.ResponseWriter, r *http.Request) {
	artists, albums, songs, err := s.searchCommon(r)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
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
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleSearch3(w http.ResponseWriter, r *http.Request) {
	artists, albums, songs, err := s.searchCommon(r)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SearchResult3 = &models.SearchResult3{
		Artist: artists,
		Album:  albums,
		Song:   songs,
	}
	s.sendResponse(w, r, resp)
}
