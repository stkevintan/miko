package subsonic

import (
	"crypto/md5"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stkevintan/miko/api/models"
	"gorm.io/gorm"
)

type Subsonic struct {
	db   *gorm.DB
	rest *gin.RouterGroup
}

func NewSubsonic(r *gin.Engine, db *gorm.DB) *Subsonic {
	return &Subsonic{
		db:   db,
		rest: r.Group("/rest"),
	}
}

func (s *Subsonic) RegisterRoutes() {
	s.rest.Use(s.subsonicAuthMiddleware())

	// System
	s.rest.GET("/ping.view", s.handlePing)
	s.rest.GET("/getLicense.view", s.handleGetLicense)

	// Browsing
	s.rest.GET("/getMusicFolders.view", s.handleNotImplemented)
	s.rest.GET("/getIndexes.view", s.handleNotImplemented)
	s.rest.GET("/getMusicDirectory.view", s.handleNotImplemented)
	s.rest.GET("/getGenres.view", s.handleNotImplemented)
	s.rest.GET("/getArtists.view", s.handleNotImplemented)
	s.rest.GET("/getArtist.view", s.handleNotImplemented)
	s.rest.GET("/getAlbum.view", s.handleNotImplemented)
	s.rest.GET("/getSong.view", s.handleNotImplemented)
	s.rest.GET("/getVideos.view", s.handleNotImplemented)
	s.rest.GET("/getVideoInfo.view", s.handleNotImplemented)
	s.rest.GET("/getNowPlaying.view", s.handleNotImplemented)

	// Search
	s.rest.GET("/search2.view", s.handleNotImplemented)
	s.rest.GET("/search3.view", s.handleNotImplemented)

	// Playlists
	s.rest.GET("/getPlaylists.view", s.handleNotImplemented)
	s.rest.GET("/getPlaylist.view", s.handleNotImplemented)

	// Lists
	s.rest.GET("/getAlbumList.view", s.handleNotImplemented)
	s.rest.GET("/getAlbumList2.view", s.handleNotImplemented)
	s.rest.GET("/getRandomSongs.view", s.handleNotImplemented)
	s.rest.GET("/getSongsByGenre.view", s.handleNotImplemented)

	// Media
	s.rest.GET("/stream.view", s.handleNotImplemented)
	s.rest.GET("/download.view", s.handleNotImplemented)
	s.rest.GET("/getCoverArt.view", s.handleNotImplemented)
	s.rest.GET("/getLyrics.view", s.handleNotImplemented)
	s.rest.GET("/getAvatar.view", s.handleNotImplemented)

	// Podcasts
	s.rest.GET("/getPodcasts.view", s.handleNotImplemented)
	s.rest.GET("/getNewestPodcasts.view", s.handleNotImplemented)

	// Radio
	s.rest.GET("/getInternetRadioStations.view", s.handleNotImplemented)

	// Bookmarks
	s.rest.GET("/getBookmarks.view", s.handleNotImplemented)
	s.rest.GET("/getPlayQueue.view", s.handleNotImplemented)

	// Sharing
	s.rest.GET("/getShares.view", s.handleNotImplemented)

	// Starred
	s.rest.GET("/getStarred.view", s.handleNotImplemented)
	s.rest.GET("/getStarred2.view", s.handleNotImplemented)

	// Info
	s.rest.GET("/getAlbumInfo.view", s.handleNotImplemented)
	s.rest.GET("/getArtistInfo.view", s.handleNotImplemented)
	s.rest.GET("/getArtistInfo2.view", s.handleNotImplemented)
	s.rest.GET("/getSimilarSongs.view", s.handleNotImplemented)
	s.rest.GET("/getSimilarSongs2.view", s.handleNotImplemented)
	s.rest.GET("/getTopSongs.view", s.handleNotImplemented)

	// Scan
	s.rest.GET("/getScanStatus.view", s.handleNotImplemented)

	// User
	s.rest.GET("/getUser.view", s.handleNotImplemented)
	s.rest.GET("/getUsers.view", s.handleNotImplemented)
}

func (s *Subsonic) sendResponse(c *gin.Context, resp *SubsonicResponse) {
	format := c.DefaultQuery("f", "xml")
	if format == "json" {
		c.JSON(http.StatusOK, gin.H{"subsonic-response": resp})
	} else {
		c.XML(http.StatusOK, resp)
	}
}

func (s *Subsonic) handleNotImplemented(c *gin.Context) {
	s.sendResponse(c, NewErrorResponse(0, "Not implemented"))
}

func (s *Subsonic) handlePing(c *gin.Context) {
	resp := NewResponse(ResponseStatusOK)
	resp.Ping = &Ping{}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleGetLicense(c *gin.Context) {
	resp := NewResponse(ResponseStatusOK)
	resp.License = &License{
		Valid: true,
		Email: "miko@example.com",
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) subsonicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Query("u")
		password := c.Query("p")
		token := c.Query("t")
		salt := c.Query("s")

		if username == "" {
			s.sendResponse(c, NewErrorResponse(10, "User not found"))
			c.Abort()
			return
		}

		var user models.User
		if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
			s.sendResponse(c, NewErrorResponse(10, "User not found"))
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
			s.sendResponse(c, NewErrorResponse(40, "Wrong username or password"))
			c.Abort()
			return
		}

		c.Set("user", user)
		c.Next()
	}
}
