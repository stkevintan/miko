package subsonic

import (
	"net/http"
	"strings"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/browser"
	"github.com/stkevintan/miko/pkg/di"
)

func (s *Subsonic) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	count := getQueryIntOrDefault(r, "count", 20)
	offset := getQueryIntOrDefault(r, "offset", 0)

	br := di.MustInvoke[*browser.Browser](r.Context())
	songs, totalHits, err := br.SearchSongs(query, count, offset)
	if err != nil {
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

func (s *Subsonic) handleSearch2(w http.ResponseWriter, r *http.Request) {
	br := di.MustInvoke[*browser.Browser](r.Context())
	query := r.URL.Query().Get("query")
	query = strings.Trim(query, " \"'")

	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	hasFolderId := err == nil

	artists, albums, songs, err := br.Search(browser.SearchOptions{
		Query:         query,
		ArtistCount:   getQueryIntOrDefault(r, "artistCount", 20),
		ArtistOffset:  getQueryIntOrDefault(r, "artistOffset", 0),
		AlbumCount:    getQueryIntOrDefault(r, "albumCount", 20),
		AlbumOffset:   getQueryIntOrDefault(r, "albumOffset", 0),
		SongCount:     getQueryIntOrDefault(r, "songCount", 20),
		SongOffset:    getQueryIntOrDefault(r, "songOffset", 0),
		MusicFolderID: musicFolderId,
		HasFolderID:   hasFolderId,
	})

	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	// Convert ArtistID3 to Artist for SearchResult2
	searchArtists := make([]models.Artist, len(artists))
	for i, a := range artists {
		searchArtists[i] = models.Artist{
			ID:             a.ID,
			Name:           a.Name,
			ArtistImageUrl: a.ArtistImageUrl,
			Starred:        a.Starred,
		}
	}

	// Convert AlbumID3 to Child for SearchResult2
	searchAlbums := make([]models.Child, len(albums))
	for i, a := range albums {
		searchAlbums[i] = models.Child{
			ID:        a.ID,
			Title:     a.Name,
			Artist:    a.Artist,
			ArtistID:  a.ArtistID,
			CoverArt:  a.CoverArt,
			IsDir:     true,
			Created:   &a.Created,
			Year:      a.Year,
			Genre:     a.Genre,
			Starred:   a.Starred,
			Duration:  a.Duration,
			PlayCount: a.PlayCount,
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
	br := di.MustInvoke[*browser.Browser](r.Context())
	query := r.URL.Query().Get("query")
	query = strings.Trim(query, " \"'")

	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	hasFolderId := err == nil

	artists, albums, songs, err := br.Search(browser.SearchOptions{
		Query:         query,
		ArtistCount:   getQueryIntOrDefault(r, "artistCount", 20),
		ArtistOffset:  getQueryIntOrDefault(r, "artistOffset", 0),
		AlbumCount:    getQueryIntOrDefault(r, "albumCount", 20),
		AlbumOffset:   getQueryIntOrDefault(r, "albumOffset", 0),
		SongCount:     getQueryIntOrDefault(r, "songCount", 20),
		SongOffset:    getQueryIntOrDefault(r, "songOffset", 0),
		MusicFolderID: musicFolderId,
		HasFolderID:   hasFolderId,
	})

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
