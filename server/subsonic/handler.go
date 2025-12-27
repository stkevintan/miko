package subsonic

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
	"gorm.io/gorm"
)

type Subsonic struct {
	injector   do.Injector
	nowPlaying sync.Map // key: string (username:clientName), value: models.NowPlayingRecord
}

func New(injector do.Injector) *Subsonic {
	return &Subsonic{
		injector:   injector,
		nowPlaying: sync.Map{},
	}
}

func (s *Subsonic) RegisterRoutes(r *gin.Engine) *gin.RouterGroup {
	rest := r.Group("/rest")
	// Handle Subsonic .view suffix by rewriting and re-routing
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/rest") && strings.HasSuffix(path, ".view") {
			c.Request.URL.Path = strings.TrimSuffix(path, ".view")
			log.Info("Rewriting Subsonic path (NoRoute): %s -> %s", path, c.Request.URL.Path)
			r.HandleContext(c)
			return
		}
	})
	rest.Use(s.subsonicAuth())

	// System
	rest.GET("/ping", s.handlePing)
	rest.GET("/getLicense", s.handleGetLicense)

	// Browsing
	rest.GET("/getMusicFolders", s.handleGetMusicFolders)
	rest.GET("/getIndexes", s.handleGetIndexes)
	rest.GET("/getMusicDirectory", s.handleGetMusicDirectory)
	rest.GET("/getGenres", s.handleGetGenres)
	rest.GET("/getArtists", s.handleGetArtists)
	rest.GET("/getArtist", s.handleGetArtist)
	rest.GET("/getAlbum", s.handleGetAlbum)
	rest.GET("/getSong", s.handleGetSong)

	rest.GET("/getVideos", s.handleUnsupported)
	rest.GET("/getVideoInfo", s.handleUnsupported)
	rest.GET("/getArtistInfo", s.handleGetArtistInfo)

	rest.GET("/getArtistInfo2", s.handleGetArtistInfo2)
	rest.GET("/getAlbumInfo", s.handleGetAlbumInfo)
	rest.GET("/getAlbumInfo2", s.handleGetAlbumInfo2)
	rest.GET("/getSimilarSongs", s.handleGetSimilarSongs)
	rest.GET("/getSimilarSongs2", s.handleGetSimilarSongs2)
	rest.GET("/getTopSongs", s.handleGetTopSongs)

	// Album/song lists
	rest.GET("/getAlbumList", s.handleGetAlbumList)
	rest.GET("/getAlbumList2", s.handleGetAlbumList2)
	rest.GET("/getRandomSongs", s.handleGetRandomSongs)
	rest.GET("/getSongsByGenre", s.handleGetSongsByGenre)
	rest.GET("/getNowPlaying", s.handleGetNowPlaying)
	rest.GET("/getStarred", s.handleGetStarred)
	rest.GET("/getStarred2", s.handleGetStarred2)

	// Searching
	rest.GET("/search", s.handleSearch)
	rest.GET("/search2", s.handleSearch2)
	rest.GET("/search3", s.handleSearch3)

	// Playlists
	rest.GET("/getPlaylists", s.handleGetPlaylists)
	rest.GET("/getPlaylist", s.handleGetPlaylist)
	rest.GET("/createPlaylist", s.handleCreatePlaylist)
	rest.GET("/updatePlaylist", s.handleUpdatePlaylist)
	rest.GET("/deletePlaylist", s.handleDeletePlaylist)

	// Media retrieval
	rest.GET("/stream", s.handleStream)
	rest.GET("/download", s.handleDownload)
	rest.GET("/hls.m3u8", s.handleUnsupported)
	rest.GET("/getCaptions", s.handleUnsupported)
	rest.GET("/getCoverArt", s.handleGetCoverArt)
	rest.GET("/getLyrics", s.handleGetLyrics)
	rest.GET("/getAvatar", s.handleGetAvatar)

	// Media annotation
	rest.GET("/star", s.handleNotImplemented)
	rest.GET("/unstar", s.handleNotImplemented)
	rest.GET("/setRating", s.handleNotImplemented)
	rest.GET("/scrobble", s.handleScrobble)

	// Sharing
	rest.GET("/getShares", s.handleNotImplemented)
	rest.GET("/createShare", s.handleNotImplemented)
	rest.GET("/updateShare", s.handleNotImplemented)
	rest.GET("/deleteShare", s.handleNotImplemented)

	// Podcast
	rest.GET("/getPodcasts", s.handleNotImplemented)
	rest.GET("/getNewestPodcasts", s.handleNotImplemented)
	rest.GET("/refreshPodcasts", s.handleNotImplemented)
	rest.GET("/createPodcastChannel", s.handleNotImplemented)
	rest.GET("/deletePodcastChannel", s.handleNotImplemented)
	rest.GET("/deletePodcastEpisode", s.handleNotImplemented)
	rest.GET("/downloadPodcastEpisode", s.handleNotImplemented)

	// Jukebox
	rest.GET("/jukeboxControl", s.handleNotImplemented)

	// Internet radio
	rest.GET("/getInternetRadioStations", s.handleNotImplemented)
	rest.GET("/createInternetRadioStation", s.handleNotImplemented)
	rest.GET("/updateInternetRadioStation", s.handleNotImplemented)
	rest.GET("/deleteInternetRadioStation", s.handleNotImplemented)

	// Chat
	rest.GET("/getChatMessages", s.handleNotImplemented)
	rest.GET("/addChatMessage", s.handleNotImplemented)

	// User management
	rest.GET("/getUser", s.handleGetUser)
	rest.GET("/getUsers", s.handleGetUsers)
	rest.GET("/createUser", s.handleNotImplemented)
	rest.GET("/updateUser", s.handleNotImplemented)
	rest.GET("/deleteUser", s.handleNotImplemented)
	rest.GET("/changePassword", s.handleNotImplemented)

	// Bookmarks
	rest.GET("/getBookmarks", s.handleNotImplemented)
	rest.GET("/createBookmark", s.handleNotImplemented)
	rest.GET("/deleteBookmark", s.handleNotImplemented)
	rest.GET("/getPlayQueue", s.handleNotImplemented)
	rest.GET("/savePlayQueue", s.handleNotImplemented)

	// Media library scanning
	rest.GET("/getScanStatus", s.handleGetScanStatus)
	rest.GET("/startScan", s.handleStartScan)

	return rest
}

func (s *Subsonic) sendResponse(c *gin.Context, resp *models.SubsonicResponse) {
	format := c.DefaultQuery("f", "xml")
	if format == "json" {
		c.JSON(http.StatusOK, gin.H{"subsonic-response": resp})
	} else {
		c.XML(http.StatusOK, resp)
	}
}

func (s *Subsonic) handleUnsupported(c *gin.Context) {
	s.sendResponse(c, models.NewErrorResponse(0, "Not supported"))
}

func (s *Subsonic) handleNotImplemented(c *gin.Context) {
	s.sendResponse(c, models.NewErrorResponse(0, "Not implemented"))
}

func (s *Subsonic) subsonicAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Query("u")
		password := c.Query("p")
		token := c.Query("t")
		salt := c.Query("s")

		if username == "" {
			s.sendResponse(c, models.NewErrorResponse(10, "User not found"))
			c.Abort()
			return
		}

		user := models.LoginRequest{}
		db := do.MustInvoke[*gorm.DB](s.injector)
		if err := db.Model(&models.User{}).Select("username", "password").Where("username = ?", username).First(&user).Error; err != nil {
			s.sendResponse(c, models.NewErrorResponse(10, "User not found"))
			c.Abort()
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
			s.sendResponse(c, models.NewErrorResponse(40, "Wrong username or password"))
			c.Abort()
			return
		}

		c.Set("Username", user.Username)
		c.Next()
	}
}
