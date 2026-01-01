package subsonic

import (
	"net/http"
	"time"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/browser"
	"github.com/stkevintan/miko/pkg/di"
	"gorm.io/gorm"
)

func (s *Subsonic) handleGetAlbumList2(w http.ResponseWriter, r *http.Request) {
	br := di.MustInvoke[*browser.Browser](r.Context())
	query := r.URL.Query()
	listType := query.Get("type")
	if listType == "" {
		listType = "newest"
	}

	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	hasFolderId := err == nil

	albums, err := br.GetAlbums(browser.AlbumListOptions{
		Type:          listType,
		Size:          getQueryIntOrDefault(r, "size", 10),
		Offset:        getQueryIntOrDefault(r, "offset", 0),
		Genre:         query.Get("genre"),
		FromYear:      getQueryIntOrDefault(r, "fromYear", 0),
		ToYear:        getQueryIntOrDefault(r, "toYear", 3000),
		MusicFolderID: musicFolderId,
		HasFolderID:   hasFolderId,
	})

	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.AlbumList2 = &models.AlbumList2{
		Album: albums,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetAlbumList(w http.ResponseWriter, r *http.Request) {
	br := di.MustInvoke[*browser.Browser](r.Context())
	query := r.URL.Query()
	listType := query.Get("type")
	if listType == "" {
		listType = "newest"
	}

	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	hasFolderId := err == nil

	albums, err := br.GetAlbums(browser.AlbumListOptions{
		Type:          listType,
		Size:          getQueryIntOrDefault(r, "size", 10),
		Offset:        getQueryIntOrDefault(r, "offset", 0),
		Genre:         query.Get("genre"),
		FromYear:      getQueryIntOrDefault(r, "fromYear", 0),
		ToYear:        getQueryIntOrDefault(r, "toYear", 3000),
		MusicFolderID: musicFolderId,
		HasFolderID:   hasFolderId,
	})

	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	// Convert AlbumID3 to Child for AlbumList (which uses Child elements)
	children := make([]models.Child, len(albums))
	for i, a := range albums {
		children[i] = models.Child{
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
	resp.AlbumList = &models.AlbumList{
		Album: children,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetNowPlaying(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())

	entries := make([]models.NowPlayingEntry, 0)

	s.nowPlaying.Range(func(key, value interface{}) bool {
		record, ok := value.(models.NowPlayingRecord)
		if !ok {
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
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetRandomSongs(w http.ResponseWriter, r *http.Request) {
	br := di.MustInvoke[*browser.Browser](r.Context())
	query := r.URL.Query()

	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	hasFolderId := err == nil

	songs, err := br.GetRandomSongs(browser.AlbumListOptions{
		Size:          getQueryIntOrDefault(r, "size", 10),
		Genre:         query.Get("genre"),
		FromYear:      getQueryIntOrDefault(r, "fromYear", 0),
		ToYear:        getQueryIntOrDefault(r, "toYear", 3000),
		MusicFolderID: musicFolderId,
		HasFolderID:   hasFolderId,
	})

	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, "Failed to fetch songs"))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.RandomSongs = &models.Songs{
		Song: songs,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetSongsByGenre(w http.ResponseWriter, r *http.Request) {
	genre := r.URL.Query().Get("genre")
	if genre == "" {
		s.sendResponse(w, r, models.NewErrorResponse(10, "Genre is required"))
		return
	}

	br := di.MustInvoke[*browser.Browser](r.Context())
	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	hasFolderId := err == nil

	songs, err := br.GetSongsByGenre(
		genre,
		getQueryIntOrDefault(r, "count", 10),
		getQueryIntOrDefault(r, "offset", 0),
		musicFolderId,
		hasFolderId,
	)

	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.SongsByGenre = &models.Songs{
		Song: songs,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) getStarredItems(r *http.Request) ([]models.ArtistID3, []models.AlbumID3, []models.Child, error) {
	br := di.MustInvoke[*browser.Browser](r.Context())
	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	hasFolderId := err == nil
	return br.GetStarredItems(musicFolderId, hasFolderId)
}

func (s *Subsonic) handleGetStarred(w http.ResponseWriter, r *http.Request) {
	artists, albums, songs, err := s.getStarredItems(r)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
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
	resp.Starred = &models.Starred{
		Artist: starredArtists,
		Album:  starredAlbums,
		Song:   songs,
	}
	s.sendResponse(w, r, resp)
}

func (s *Subsonic) handleGetStarred2(w http.ResponseWriter, r *http.Request) {
	artists, albums, songs, err := s.getStarredItems(r)
	if err != nil {
		s.sendResponse(w, r, models.NewErrorResponse(0, err.Error()))
		return
	}

	resp := models.NewResponse(models.ResponseStatusOK)
	resp.Starred2 = &models.Starred2{
		Artist: artists,
		Album:  albums,
		Song:   songs,
	}
	s.sendResponse(w, r, resp)
}
