package scanner

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/tags"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type imageTask struct {
	path     string
	coverArt string
}

type worker struct {
	seenArtists map[string]bool
	seenGenres  map[string]bool
	seenAlbums  map[string]bool
	imageTasks  chan imageTask
	cacheDir    string
	db          *gorm.DB
}

func (s *Scanner) saveResults(resultChan <-chan scanResult, cacheDir string) {
	w := &worker{
		seenArtists: make(map[string]bool),
		seenGenres:  make(map[string]bool),
		seenAlbums:  make(map[string]bool),
		imageTasks:  make(chan imageTask, s.numWorkers*10),
		cacheDir:    cacheDir,
		db:          s.db,
	}

	var imageWg sync.WaitGroup
	w.startImageWorkers(&imageWg, s.numWorkers)

	var children []models.Child
	flushChildren := func() {
		if len(children) > 0 {
			s.db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(children, 100)
			children = children[:0]
		}
	}

	for res := range resultChan {
		child := res.child
		if res.tags != nil {
			w.processMetadata(child, res.tags, res.path)
		}

		children = append(children, *child)
		if len(children) >= 100 {
			flushChildren()
		}
		if !child.IsDir {
			s.scanCount.Add(1)
		}
	}

	flushChildren()
	close(w.imageTasks)
	imageWg.Wait()
}

func (s *Scanner) SaveCoverArt(coverArt string, data []byte) error {
	if coverArt == "" {
		return nil
	}
	cacheDir := GetCoverCacheDir(s.cfg)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}
	cachePath := filepath.Join(cacheDir, coverArt)
	return os.WriteFile(cachePath, data, 0644)
}

func (w *worker) startImageWorkers(wg *sync.WaitGroup, numWorkers int) {
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range w.imageTasks {
				if t.coverArt == "" {
					continue
				}
				p := filepath.Join(w.cacheDir, t.coverArt)
				// if p exists, skip
				if _, err := os.Stat(p); err == nil {
					continue
				}
				img, err := tags.ReadImage(t.path)
				if err != nil || len(img) == 0 {
					continue
				}
				if err := os.WriteFile(p, img, 0644); err != nil {
					log.Warn("Failed to write cover art to cache for %s: %v", t.coverArt, err)
				}
			}
		}()
	}
}

func (w *worker) processMetadata(child *models.Child, t *tags.Tags, path string) {
	if t.Title != "" {
		child.Title = t.Title
	}
	if t.Artist != "" {
		child.Artist = t.Artist
		child.Artists = w.getArtistsFromNames(t.Artists)
		if len(child.Artists) > 0 {
			child.ArtistID = child.Artists[0].ID
		}
	}
	if t.Album != "" {
		child.Album = t.Album
	}
	child.Track = t.Track
	child.DiscNumber = t.Disc
	child.Year = t.Year
	if t.Genre != "" {
		child.Genre = t.Genre
		child.Genres = w.getGenresFromNames(t.Genres)
	}
	if t.Lyrics != "" {
		child.Lyrics = t.Lyrics
	}
	child.Duration = t.Duration
	child.BitRate = t.Bitrate

	// Album logic
	if child.Album != "" {
		w.handleAlbum(child, t, path)
	} else {
		// for child without album, use its own cover art
		child.CoverArt = child.ID
		w.imageTasks <- imageTask{path: path, coverArt: child.CoverArt}
	}
}

func (w *worker) handleAlbum(child *models.Child, t *tags.Tags, path string) {
	albumArtistStr := t.AlbumArtist
	var albumArtists []models.ArtistID3
	if albumArtistStr != "" {
		albumArtists = w.getArtistsFromNames(t.AlbumArtists)
	}

	groupArtist := child.Artist
	groupArtists := child.Artists
	if albumArtistStr != "" {
		groupArtist = albumArtistStr
		groupArtists = albumArtists
	}

	displayArtist := groupArtist
	if displayArtist == "" {
		displayArtist = "Unknown Artist"
	}

	albumID := GenerateAlbumID(displayArtist, child.Album)
	child.AlbumID = albumID
	child.CoverArt = "al-" + albumID

	// Create the album DB entry only once
	if !w.seenAlbums[albumID] {
		created := time.Now()
		if child.Created != nil {
			created = *child.Created
		}
		album := models.AlbumID3{
			ID:       albumID,
			Name:     child.Album,
			Artist:   displayArtist,
			Created:  created,
			CoverArt: child.CoverArt,
		}
		if len(groupArtists) > 0 {
			album.ArtistID = groupArtists[0].ID
			album.Artists = groupArtists
		}

		w.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&album)
		w.seenAlbums[albumID] = true
	}

	// Always queue an imageTask to attempt to cache cover art from this song.
	// The worker will skip if the cached file already exists, so this is efficient
	// and ensures cover art can be found from any song in the album.
	w.imageTasks <- imageTask{path: path, coverArt: child.CoverArt}
}

func (w *worker) getArtistsFromNames(names []string) []models.ArtistID3 {
	var artists []models.ArtistID3
	for _, name := range names {
		artistID := GenerateArtistID(name)
		artist := models.ArtistID3{
			ID:       artistID,
			Name:     name,
			CoverArt: "ar-" + artistID,
		}
		if !w.seenArtists[artistID] {
			w.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&artist)
			w.seenArtists[artistID] = true
		}
		artists = append(artists, artist)
	}
	return artists
}

func (w *worker) getGenresFromNames(names []string) []models.Genre {
	var genres []models.Genre
	for _, name := range names {
		genre := models.Genre{Name: name}
		if !w.seenGenres[name] {
			w.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&genre)
			w.seenGenres[name] = true
		}
		genres = append(genres, genre)
	}
	return genres
}
