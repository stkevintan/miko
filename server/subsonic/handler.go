package subsonic

import (
	"crypto/md5"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

type Subsonic struct {
	injector do.Injector
}

func New(injector do.Injector) *Subsonic {
	return &Subsonic{
		injector: injector,
	}
}

func (s *Subsonic) RegisterRoutes(r *gin.Engine) *gin.RouterGroup {
	rest := r.Group("/rest")
	rest.Use(s.subsonicAuthMiddleware())

	// System
	rest.GET("/ping.view", s.handlePing)
	rest.GET("/getLicense.view", s.handleGetLicense)

	// Browsing
	rest.GET("/getMusicFolders.view", s.handleGetMusicFolders)
	rest.GET("/getIndexes.view", s.handleGetIndexes)
	rest.GET("/getMusicDirectory.view", s.handleGetMusicDirectory)
	rest.GET("/getGenres.view", s.handleGetGenres)
	rest.GET("/getArtists.view", s.handleGetArtists)
	rest.GET("/getArtist.view", s.handleGetArtist)
	rest.GET("/getAlbum.view", s.handleGetAlbum)
	rest.GET("/getSong.view", s.handleGetSong)
	rest.GET("/getVideos.view", s.handleNotImplemented)
	rest.GET("/getVideoInfo.view", s.handleNotImplemented)
	rest.GET("/getNowPlaying.view", s.handleNotImplemented)

	// Search
	rest.GET("/search2.view", s.handleNotImplemented)
	rest.GET("/search3.view", s.handleNotImplemented)

	// Playlists
	rest.GET("/getPlaylists.view", s.handleNotImplemented)
	rest.GET("/getPlaylist.view", s.handleNotImplemented)

	// Lists
	rest.GET("/getAlbumList.view", s.handleNotImplemented)
	rest.GET("/getAlbumList2.view", s.handleNotImplemented)
	rest.GET("/getRandomSongs.view", s.handleNotImplemented)
	rest.GET("/getSongsByGenre.view", s.handleNotImplemented)

	// Media
	rest.GET("/stream.view", s.handleNotImplemented)
	rest.GET("/download.view", s.handleNotImplemented)
	rest.GET("/getCoverArt.view", s.handleNotImplemented)
	rest.GET("/getLyrics.view", s.handleNotImplemented)
	rest.GET("/getAvatar.view", s.handleNotImplemented)

	// Podcasts
	rest.GET("/getPodcasts.view", s.handleNotImplemented)
	rest.GET("/getNewestPodcasts.view", s.handleNotImplemented)

	// Radio
	rest.GET("/getInternetRadioStations.view", s.handleNotImplemented)

	// Bookmarks
	rest.GET("/getBookmarks.view", s.handleNotImplemented)
	rest.GET("/getPlayQueue.view", s.handleNotImplemented)

	// Sharing
	rest.GET("/getShares.view", s.handleNotImplemented)

	// Starred
	rest.GET("/getStarred.view", s.handleNotImplemented)
	rest.GET("/getStarred2.view", s.handleNotImplemented)

	// Info
	rest.GET("/getAlbumInfo.view", s.handleNotImplemented)
	rest.GET("/getArtistInfo.view", s.handleNotImplemented)
	rest.GET("/getArtistInfo2.view", s.handleNotImplemented)
	rest.GET("/getSimilarSongs.view", s.handleNotImplemented)
	rest.GET("/getSimilarSongs2.view", s.handleNotImplemented)
	rest.GET("/getTopSongs.view", s.handleNotImplemented)

	// Scan
	rest.GET("/getScanStatus.view", s.handleGetScanStatus)
	rest.GET("/startScan.view", s.handleStartScan)

	// User
	rest.GET("/getUser.view", s.handleNotImplemented)
	rest.GET("/getUsers.view", s.handleNotImplemented)
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

func (s *Subsonic) handleNotImplemented(c *gin.Context) {
	s.sendResponse(c, models.NewErrorResponse(0, "Not implemented"))
}

func (s *Subsonic) subsonicAuthMiddleware() gin.HandlerFunc {
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

		var user models.User
		db := do.MustInvoke[*gorm.DB](s.injector)
		if err := db.Where("username = ?", username).First(&user).Error; err != nil {
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

		c.Set("user", user)
		c.Next()
	}
}
