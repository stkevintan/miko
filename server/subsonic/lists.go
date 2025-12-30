package subsonic

import (
	"net/http"
	"time"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"gorm.io/gorm"
)

func getAlbums(r *http.Request) ([]models.AlbumID3, error) {
	query := r.URL.Query()
	listType := query.Get("type")
	if listType == "" {
		listType = "newest"
	}
	size := getQueryIntOrDefault(r, "size", 10)
	offset := getQueryIntOrDefault(r, "offset", 0)
	genre := query.Get("genre")
	fromYear := getQueryIntOrDefault(r, "fromYear", 0)
	toYear := getQueryIntOrDefault(r, "toYear", 3000)

	db := di.MustInvoke[*gorm.DB](r.Context())
	var albums []models.AlbumID3

	dbQuery := db.Scopes(models.AlbumWithStats(listType == "recent")).Limit(size).Offset(offset)

	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	if err == nil {
		dbQuery = dbQuery.Joins("JOIN children ON children.album_id = album_id3.id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("album_id3.id")
	}

	/**
	The list type. Must be one of the following: random, newest, frequent, recent, starred, alphabeticalByName or alphabeticalByArtist
	Since 1.10.1 you can use byYear and byGenre to list albums in a given year range or genre
	*/
	switch listType {
	case "random":
		dbQuery = dbQuery.Order("RANDOM()")
	case "newest":
		dbQuery = dbQuery.Order("album_id3.created DESC")
	case "frequent":
		dbQuery = dbQuery.Order("play_count DESC")
	case "recent":
dbQuery = dbQuery.Having("last_played IS NOT NULL").
			Order("last_played DESC")
	case "starred":
		dbQuery = dbQuery.Where("album_id3.starred IS NOT NULL").Order("album_id3.starred DESC")
	case "alphabeticalByName":
		dbQuery = dbQuery.Order("album_id3.name ASC")
	case "alphabeticalByArtist":
		dbQuery = dbQuery.Order("album_id3.artist ASC")
	case "byYear":
		dbQuery = dbQuery.Where("album_id3.year >= ? AND album_id3.year <= ?", fromYear, toYear).Order("album_id3.year DESC")
	case "byGenre":
		if genre != "" {
			dbQuery = dbQuery.Joins("JOIN album_genres ON album_genres.album_id3_id = album_id3.id").
				Where("album_genres.genre_name = ?", genre)
		}
	default:
		dbQuery = dbQuery.Order("album_id3.created DESC")
	}

	err = dbQuery.Find(&albums).Error
	return albums, err
}

func (s *Subsonic) handleGetAlbumList2(w http.ResponseWriter, r *http.Request) {
	albums, err := getAlbums(r)
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
	// getAlbumList is the older version, we can just wrap handleGetAlbumList2
	// but it returns SearchResult instead of AlbumList2 in some versions?
	// Actually it returns <albumList> which is same as <albumList2> but with different element names.
	// For simplicity, let's just use the same logic but return AlbumList.

	albums, err := getAlbums(r)
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

func (s *Subsonic) handleGetRandomSongs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	size := getQueryIntOrDefault(r, "size", 10)
	genre := query.Get("genre")
	fromYear := getQueryIntOrDefault(r, "fromYear", 0)
	toYear := getQueryIntOrDefault(r, "toYear", 3000)

	db := di.MustInvoke[*gorm.DB](r.Context())
	var songs []models.Child

	dbQuery := db.Where("is_dir = ?", false).Limit(size).Order("RANDOM()")

	// Optional musicFolderId filter
	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	if err == nil {
		dbQuery = dbQuery.Where("music_folder_id = ?", musicFolderId)
	}

	if genre != "" {
		dbQuery = dbQuery.Joins("JOIN song_genres ON song_genres.child_id = children.id").
			Where("song_genres.genre_name = ?", genre)
	}
	if fromYear > 0 {
		dbQuery = dbQuery.Where("children.year >= ?", fromYear)
	}
	if toYear < 3000 {
		dbQuery = dbQuery.Where("children.year <= ?", toYear)
	}

	if err := dbQuery.Find(&songs).Error; err != nil {
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

	count := getQueryIntOrDefault(r, "count", 10)
	offset := getQueryIntOrDefault(r, "offset", 0)

	db := di.MustInvoke[*gorm.DB](r.Context())
	var songs []models.Child
	dbQuery := db.Joins("JOIN song_genres ON song_genres.child_id = children.id").
		Where("song_genres.genre_name = ?", genre)

	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
	if err == nil {
		dbQuery = dbQuery.Where("children.music_folder_id = ?", musicFolderId)
	}

	err = dbQuery.Limit(count).Offset(offset).Find(&songs).Error

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

func (s *Subsonic) handleGetNowPlaying(w http.ResponseWriter, r *http.Request) {
	db := di.MustInvoke[*gorm.DB](r.Context())

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
	s.sendResponse(w, r, resp)
}

func getStarredItems(r *http.Request) ([]models.ArtistID3, []models.AlbumID3, []models.Child, error) {
	db := di.MustInvoke[*gorm.DB](r.Context())

	var artists []models.ArtistID3
	artistQuery := db.Scopes(models.ArtistWithStats).Where("artist_id3.starred IS NOT NULL")
	musicFolderId, err := getQueryInt[uint](r, "musicFolderId")
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
	albumQuery := db.Scopes(models.AlbumWithStats(false)).Where("album_id3.starred IS NOT NULL")
	if musicFolderExists {
		albumQuery = albumQuery.Joins("JOIN children ON children.album_id = album_id3.id").
			Where("children.music_folder_id = ?", musicFolderId).
			Group("album_id3.id")
	}
	if err := albumQuery.Find(&albums).Error; err != nil {
		return nil, nil, nil, err
	}

	var songs []models.Child
	songQuery := db.Where("children.is_dir = ? AND children.starred IS NOT NULL", false)
	if musicFolderExists {
		songQuery = songQuery.Where("children.music_folder_id = ?", musicFolderId)
	}
	if err := songQuery.Find(&songs).Error; err != nil {
		return nil, nil, nil, err
	}

	return artists, albums, songs, nil
}

func (s *Subsonic) handleGetStarred(w http.ResponseWriter, r *http.Request) {
	artists, albums, songs, err := getStarredItems(r)
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
	artists, albums, songs, err := getStarredItems(r)
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
