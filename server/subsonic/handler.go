package subsonic

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/di"
	"gorm.io/gorm"
)

type Subsonic struct {
	ctx        context.Context
	nowPlaying sync.Map // key: string (username:clientName), value: models.NowPlayingRecord
}

func New(ctx context.Context) *Subsonic {
	return &Subsonic{
		ctx:        ctx,
		nowPlaying: sync.Map{},
	}
}

func (s *Subsonic) RegisterRoutes(r chi.Router) {
	r.Route("/rest", func(r chi.Router) {
		r.Use(s.subsonicAuth)

		// System
		r.Get("/ping", s.handlePing)
		r.Get("/getLicense", s.handleGetLicense)
		r.Get("/getOpenSubsonicExtensions", s.handleGetOpenSubsonicExtensions)

		// Browsing
		r.Get("/getMusicFolders", s.handleGetMusicFolders)
		r.Get("/getIndexes", s.handleGetIndexes)
		r.Get("/getMusicDirectory", s.handleGetMusicDirectory)
		r.Get("/getGenres", s.handleGetGenres)
		r.Get("/getArtists", s.handleGetArtists)
		r.Get("/getArtist", s.handleGetArtist)
		r.Get("/getAlbum", s.handleGetAlbum)
		r.Get("/getSong", s.handleGetSong)

		r.Get("/getVideos", s.handleUnsupported)
		r.Get("/getVideoInfo", s.handleUnsupported)
		r.Get("/getArtistInfo", s.handleGetArtistInfo)

		r.Get("/getArtistInfo2", s.handleGetArtistInfo2)
		r.Get("/getAlbumInfo", s.handleGetAlbumInfo)
		r.Get("/getAlbumInfo2", s.handleGetAlbumInfo2)
		r.Get("/getSimilarSongs", s.handleGetSimilarSongs)
		r.Get("/getSimilarSongs2", s.handleGetSimilarSongs2)
		r.Get("/getTopSongs", s.handleGetTopSongs)

		// Album/song lists
		r.Get("/getAlbumList", s.handleGetAlbumList)
		r.Get("/getAlbumList2", s.handleGetAlbumList2)
		r.Get("/getRandomSongs", s.handleGetRandomSongs)
		r.Get("/getSongsByGenre", s.handleGetSongsByGenre)
		r.Get("/getNowPlaying", s.handleGetNowPlaying)
		r.Get("/getStarred", s.handleGetStarred)
		r.Get("/getStarred2", s.handleGetStarred2)

		// Searching
		r.Get("/search", s.handleSearch)
		r.Get("/search2", s.handleSearch2)
		r.Get("/search3", s.handleSearch3)

		// Playlists
		r.Get("/getPlaylists", s.handleGetPlaylists)
		r.Get("/getPlaylist", s.handleGetPlaylist)
		r.Get("/createPlaylist", s.handleCreatePlaylist)
		r.Get("/updatePlaylist", s.handleUpdatePlaylist)
		r.Get("/deletePlaylist", s.handleDeletePlaylist)

		// Media retrieval
		r.Get("/stream", s.handleStream)
		r.Get("/download", s.handleDownload)
		r.Get("/hls.m3u8", s.handleUnsupported)
		r.Get("/getCaptions", s.handleUnsupported)
		r.Get("/getCoverArt", s.handleGetCoverArt)
		r.Get("/getLyrics", s.handleGetLyrics)
		r.Get("/getLyricsBySongId", s.handleGetLyricsBySongId)
		r.Get("/getAvatar", s.handleGetAvatar)
		r.Head("/getAvatar", s.handleGetAvatar)

		// Media annotation
		r.Get("/star", s.handleStar)
		r.Get("/unstar", s.handleUnstar)
		r.Get("/setRating", s.handleSetRating)
		r.Get("/scrobble", s.handleScrobble)

		// Sharing
		r.Get("/getShares", s.handleNotImplemented)
		r.Get("/createShare", s.handleNotImplemented)
		r.Get("/updateShare", s.handleNotImplemented)
		r.Get("/deleteShare", s.handleNotImplemented)

		// Podcast
		r.Get("/getPodcasts", s.handleNotImplemented)
		r.Get("/getNewestPodcasts", s.handleNotImplemented)
		r.Get("/refreshPodcasts", s.handleNotImplemented)
		r.Get("/createPodcastChannel", s.handleNotImplemented)
		r.Get("/deletePodcastChannel", s.handleNotImplemented)
		r.Get("/deletePodcastEpisode", s.handleNotImplemented)
		r.Get("/downloadPodcastEpisode", s.handleNotImplemented)

		// Jukebox
		r.Get("/jukeboxControl", s.handleNotImplemented)

		// Internet radio
		r.Get("/getInternetRadioStations", s.handleNotImplemented)
		r.Get("/createInternetRadioStation", s.handleNotImplemented)
		r.Get("/updateInternetRadioStation", s.handleNotImplemented)
		r.Get("/deleteInternetRadioStation", s.handleNotImplemented)

		// Chat
		r.Get("/getChatMessages", s.handleNotImplemented)
		r.Get("/addChatMessage", s.handleNotImplemented)

		// User management
		r.Get("/getUser", s.handleGetUser)
		r.Get("/getUsers", s.handleGetUsers)
		r.Get("/createUser", s.handleNotImplemented)
		r.Get("/updateUser", s.handleNotImplemented)
		r.Get("/deleteUser", s.handleNotImplemented)
		r.Get("/changePassword", s.handleNotImplemented)

		// Bookmarks
		r.Get("/getBookmarks", s.handleNotImplemented)
		r.Get("/createBookmark", s.handleNotImplemented)
		r.Get("/deleteBookmark", s.handleNotImplemented)
		r.Get("/getPlayQueue", s.handleNotImplemented)
		r.Get("/savePlayQueue", s.handleNotImplemented)

		// Media library scanning
		r.Get("/getScanStatus", s.handleGetScanStatus)
		r.Get("/startScan", s.handleStartScan)
	})
}

func (s *Subsonic) sendResponse(w http.ResponseWriter, r *http.Request, resp *models.SubsonicResponse) {
	format := r.URL.Query().Get("f")
	if format == "" {
		format = "xml"
	}

	if format == "json" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"subsonic-response": resp})
	} else {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, xml.Header)
		xml.NewEncoder(w).Encode(resp)
	}
}

func (s *Subsonic) handleUnsupported(w http.ResponseWriter, r *http.Request) {
	s.sendResponse(w, r, models.NewErrorResponse(0, "Not supported"))
}

func (s *Subsonic) handleNotImplemented(w http.ResponseWriter, r *http.Request) {
	s.sendResponse(w, r, models.NewErrorResponse(0, "Not implemented"))
}

func (s *Subsonic) subsonicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		username := query.Get("u")
		password := query.Get("p")
		token := query.Get("t")
		salt := query.Get("s")

		if username == "" {
			s.sendResponse(w, r, models.NewErrorResponse(10, "User not found"))
			return
		}

		user := models.LoginRequest{}
		db := di.MustInvoke[*gorm.DB](s.ctx)
		if err := db.Model(&models.User{}).Select("username", "password").Where("username = ?", username).First(&user).Error; err != nil {
			s.sendResponse(w, r, models.NewErrorResponse(10, "User not found"))
			return
		}

		authenticated := false
		if password != "" {
			// Clear text password auth
			if user.Password == password {
				authenticated = true
			}
		} else if token != "" && salt != "" {
			// Token auth: t = md5(password + salt)
			expectedToken := fmt.Sprintf("%x", md5.Sum([]byte(user.Password+salt)))
			if expectedToken == token {
				authenticated = true
			}
		}

		if !authenticated {
			s.sendResponse(w, r, models.NewErrorResponse(40, "Wrong username or password"))
			return
		}

		ctx := context.WithValue(r.Context(), models.UsernameKey, username)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
