package subsonic

import (
	"crypto/md5"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
	"go.senan.xyz/taglib"
	"gorm.io/gorm"
)

var (
	scanMutex  sync.Mutex
	isScanning bool
	scanCount  int64
)

func (s *Subsonic) handleGetScanStatus(c *gin.Context) {
	resp := models.NewResponse(models.ResponseStatusOK)
	resp.ScanStatus = &models.ScanStatus{
		Scanning: isScanning,
		Count:    scanCount,
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
	scanMutex.Lock()
	if isScanning {
		scanMutex.Unlock()
		return
	}
	isScanning = true
	scanCount = 0
	scanMutex.Unlock()

	defer func() {
		scanMutex.Lock()
		isScanning = false
		scanMutex.Unlock()
	}()

	db := do.MustInvoke[*gorm.DB](s.injector)
	cfg := do.MustInvoke[*config.Config](s.injector)

	for _, rootPath := range cfg.Subsonic.Folders {
		var folder models.MusicFolder
		db.Where(models.MusicFolder{Path: rootPath}).Attrs(models.MusicFolder{Name: filepath.Base(rootPath)}).FirstOrCreate(&folder)

		filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			relPath, _ := filepath.Rel(rootPath, path)
			if relPath == "." {
				return nil
			}

			// Use rootPath + relPath to avoid collisions between different music folders
			id := fmt.Sprintf("%x", md5.Sum([]byte(rootPath+relPath)))
			parentID := ""
			if parent := filepath.Dir(relPath); parent != "." {
				parentID = fmt.Sprintf("%x", md5.Sum([]byte(rootPath+parent)))
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
				db.Save(&child)
			} else {
				// Check if it's a music file
				ext := strings.ToLower(filepath.Ext(path))
				if ext != ".mp3" && ext != ".flac" && ext != ".m4a" && ext != ".wav" {
					return nil
				}

				info, _ := d.Info()
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
				}

				// Extract tags
				if tags, err := taglib.ReadTags(path); err == nil {
					if t, ok := tags[taglib.Title]; ok && len(t) > 0 {
						child.Title = t[0]
					}
					if a, ok := tags[taglib.Artist]; ok && len(a) > 0 {
						child.Artist = strings.Join(a, "; ")
						for _, ar := range a {
							artist := models.ArtistID3{
								ID:   fmt.Sprintf("%x", md5.Sum([]byte(ar))),
								Name: ar,
							}
							db.Save(&artist)
							child.Artists = append(child.Artists, artist)
						}
						child.ArtistID = child.Artists[0].ID
					}
					if al, ok := tags[taglib.Album]; ok && len(al) > 0 {
						child.Album = al[0]
					}
					if tr, ok := tags[taglib.TrackNumber]; ok && len(tr) > 0 {
						child.Track, _ = strconv.Atoi(tr[0])
					}
					if y, ok := tags[taglib.Date]; ok && len(y) > 0 {
						child.Year, _ = strconv.Atoi(y[0])
					}
					if g, ok := tags[taglib.Genre]; ok && len(g) > 0 {
						child.Genre = strings.Join(g, "; ")
						for _, genName := range g {
							genre := models.Genre{Name: genName}
							db.Save(&genre)
							child.Genres = append(child.Genres, genre)
						}
					}

					// Extract properties
					if props, err := taglib.ReadProperties(path); err == nil {
						child.Duration = int(props.Length.Seconds())
						child.BitRate = int(props.Bitrate)
					}

					// Create/Update Album
					if child.Album != "" {
						// Try to find Album Artist
						var albumArtists []models.ArtistID3
						albumArtistStr := ""
						if aa, ok := tags["ALBUMARTIST"]; ok && len(aa) > 0 {
							albumArtistStr = strings.Join(aa, "; ")
							for _, name := range aa {
								artist := models.ArtistID3{
									ID:   fmt.Sprintf("%x", md5.Sum([]byte(name))),
									Name: name,
								}
								db.Save(&artist)
								albumArtists = append(albumArtists, artist)
							}
						} else if aa, ok := tags["ALBUM ARTIST"]; ok && len(aa) > 0 {
							albumArtistStr = strings.Join(aa, "; ")
							for _, name := range aa {
								artist := models.ArtistID3{
									ID:   fmt.Sprintf("%x", md5.Sum([]byte(name))),
									Name: name,
								}
								db.Save(&artist)
								albumArtists = append(albumArtists, artist)
							}
						}

						groupArtist := child.Artist
						groupArtists := child.Artists
						if albumArtistStr != "" {
							groupArtist = albumArtistStr
							groupArtists = albumArtists
						}

						albumID := fmt.Sprintf("%x", md5.Sum([]byte(groupArtist+child.Album)))
						child.AlbumID = albumID

						album := models.AlbumID3{
							ID:       albumID,
							Name:     child.Album,
							Artist:   groupArtist,
							ArtistID: groupArtists[0].ID,
							Artists:  groupArtists,
							Created:  time.Now(),
						}
						db.Save(&album)
					}
				}

				db.Save(&child)
				scanMutex.Lock()
				scanCount++
				scanMutex.Unlock()
			}
			return nil
		})
	}
	log.Info("Scan completed. Total files: %d", scanCount)
}
