package subsonic

import (
	"crypto/md5"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
	"go.senan.xyz/taglib"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	isScanning   atomic.Bool
	scanCount    atomic.Int64
	lastScanTime atomic.Int64
)

func (s *Subsonic) handleGetScanStatus(c *gin.Context) {
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ScanStatus = &models.ScanStatus{
		Scanning: isScanning.Load(),
		Count:    scanCount.Load(),
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) handleStartScan(c *gin.Context) {
	go s.scan()
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ScanStatus = &models.ScanStatus{
		Scanning: true,
	}
	s.sendResponse(c, resp)
}

func (s *Subsonic) scan() {
	if !isScanning.CompareAndSwap(false, true) {
		return
	}
	scanCount.Store(0)

	defer isScanning.Store(false)

	db := do.MustInvoke[*gorm.DB](s.injector)
	cfg := do.MustInvoke[*config.Config](s.injector)

	cacheDir := filepath.Join(cfg.Subsonic.CacheDir, "covers")
	os.MkdirAll(cacheDir, 0755)

	seenArtists := make(map[string]bool)
	seenGenres := make(map[string]bool)
	seenAlbums := make(map[string]bool)

	var children []models.Child
	flushChildren := func() {
		if len(children) > 0 {
			db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(children, 100)
			children = children[:0]
		}
	}

	for _, rootPath := range cfg.Subsonic.Folders {
		var folder models.MusicFolder
		db.Where(models.MusicFolder{Path: rootPath}).Attrs(models.MusicFolder{Name: filepath.Base(rootPath)}).FirstOrCreate(&folder)

		filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				log.Warn("Error accessing path %q: %v", path, err)
				return nil
			}

			relPath, _ := filepath.Rel(rootPath, path)

			// Use rootPath + relPath to avoid collisions between different music folders
			id := fmt.Sprintf("%x", md5.Sum([]byte(rootPath+relPath)))
			parentID := ""
			if relPath != "." {
				if parent := filepath.Dir(relPath); parent != "." {
					parentID = fmt.Sprintf("%x", md5.Sum([]byte(rootPath+parent)))
				} else {
					parentID = fmt.Sprintf("%x", md5.Sum([]byte(rootPath+".")))
				}
			}

			if d.IsDir() {
				child := models.Child{
					ID:            id,
					Parent:        parentID,
					IsDir:         true,
					Title:         d.Name(),
					Path:          path,
					MusicFolderID: folder.ID,
				}
				children = append(children, child)
				if len(children) >= 100 {
					flushChildren()
				}
			} else {
				// Check if it's a music file
				ext := strings.ToLower(filepath.Ext(path))
				if ext != ".mp3" && ext != ".flac" && ext != ".m4a" && ext != ".wav" {
					return nil
				}

				info, err := d.Info()
				if err != nil {
					log.Warn("Failed to get file info for %q: %v", path, err)
					return nil
				}
				modTime := info.ModTime()
				child := models.Child{
					ID:            id,
					Parent:        parentID,
					IsDir:         false,
					Title:         d.Name(),
					Path:          path,
					Size:          info.Size(),
					Suffix:        ext[1:],
					ContentType:   "audio/" + ext[1:],
					MusicFolderID: folder.ID,
					Created:       &modTime,
				}

				// Extract tags
				if tags, err := taglib.ReadTags(path); err == nil {
					if t, ok := tags[taglib.Title]; ok && len(t) > 0 {
						child.Title = t[0]
					}
					if a, ok := tags[taglib.Artist]; ok && len(a) > 0 {
						child.Artist = strings.Join(a, "; ")
						child.Artists = s.getArtistsFromNames(db, a, seenArtists)
						if len(child.Artists) > 0 {
							child.ArtistID = child.Artists[0].ID
						}
					}
					if al, ok := tags[taglib.Album]; ok && len(al) > 0 {
						child.Album = al[0]
					}
					if tr, ok := tags[taglib.TrackNumber]; ok && len(tr) > 0 {
						child.Track, _ = strconv.Atoi(tr[0])
					}
					if dn, ok := tags[taglib.DiscNumber]; ok && len(dn) > 0 {
						child.DiscNumber, _ = strconv.Atoi(dn[0])
					}
					if y, ok := tags[taglib.Date]; ok && len(y) > 0 {
						child.Year, _ = strconv.Atoi(y[0])
					}
					if g, ok := tags[taglib.Genre]; ok && len(g) > 0 {
						child.Genre = strings.Join(g, "; ")
						child.Genres = s.getGenresFromNames(db, g, seenGenres)
					}

					// Extract lyrics
					if l, ok := tags[taglib.Lyrics]; ok && len(l) > 0 {
						child.Lyrics = l[0]
					} else if l, ok := tags["UNSYNCEDLYRICS"]; ok && len(l) > 0 {
						child.Lyrics = l[0]
					}

					// Extract properties
					if props, err := taglib.ReadProperties(path); err == nil {
						child.Duration = int(props.Length.Seconds())
						child.BitRate = int(props.Bitrate)
					}

					// Extract image data
					imgData, _ := taglib.ReadImage(path)

					// Create/Update Album
					if child.Album != "" {
						// Try to find Album Artist
						var albumArtists []models.ArtistID3
						albumArtistStr := ""

						aa, ok := tags[taglib.AlbumArtist]
						if !ok || len(aa) == 0 {
							aa, ok = tags["ALBUM ARTIST"]
						}

						if ok && len(aa) > 0 {
							albumArtistStr = strings.Join(aa, "; ")
							albumArtists = s.getArtistsFromNames(db, aa, seenArtists)
						}

						groupArtist := child.Artist
						groupArtists := child.Artists
						if albumArtistStr != "" {
							groupArtist = albumArtistStr
							groupArtists = albumArtists
						}

						// Use "Unknown Artist" for albums without artist information
						displayArtist := groupArtist
						if displayArtist == "" {
							displayArtist = "Unknown Artist"
						}

						// Create album ID with delimiter to avoid collisions
						albumID := fmt.Sprintf("%x", md5.Sum([]byte(displayArtist+"|"+child.Album)))
						child.AlbumID = albumID

						if !seenAlbums[albumID] {
							album := models.AlbumID3{
								ID:      albumID,
								Name:    child.Album,
								Artist:  displayArtist,
								Created: modTime,
							}
							// Set ArtistID and Artists only if we have at least one artist
							if len(groupArtists) > 0 {
								album.ArtistID = groupArtists[0].ID
								album.Artists = groupArtists
							}
							// Use song's cover art for album if available
							if len(imgData) > 0 {
								album.CoverArt = album.ID
								os.WriteFile(filepath.Join(cacheDir, album.ID), imgData, 0644)
							}
							db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&album)
							seenAlbums[albumID] = true
						}

						// If album has a cover (either just created or already in cache), use it for the child
						if _, err := os.Stat(filepath.Join(cacheDir, albumID)); err == nil {
							child.CoverArt = albumID
						}
					}

					// If still no cover art (not in album or album has no cover), use song's own if it has one
					if child.CoverArt == "" && len(imgData) > 0 {
						child.CoverArt = child.ID
						os.WriteFile(filepath.Join(cacheDir, child.ID), imgData, 0644)
					}
				}

				children = append(children, child)
				if len(children) >= 100 {
					flushChildren()
				}
				scanCount.Add(1)
			}
			return nil
		})
		flushChildren()
	}
	lastScanTime.Store(time.Now().Unix())
	log.Info("Scan completed. Total files: %d", scanCount.Load())
}

func (s *Subsonic) getArtistsFromNames(db *gorm.DB, names []string, seen map[string]bool) []models.ArtistID3 {
	var artists []models.ArtistID3
	for _, name := range names {
		artistID := fmt.Sprintf("%x", md5.Sum([]byte(name)))
		artist := models.ArtistID3{
			ID:   artistID,
			Name: name,
		}
		if !seen[artistID] {
			db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&artist)
			seen[artistID] = true
		}
		artists = append(artists, artist)
	}
	return artists
}

func (s *Subsonic) getGenresFromNames(db *gorm.DB, names []string, seen map[string]bool) []models.Genre {
	var genres []models.Genre
	for _, name := range names {
		genre := models.Genre{Name: name}
		if !seen[name] {
			db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&genre)
			seen[name] = true
		}
		genres = append(genres, genre)
	}
	return genres
}
