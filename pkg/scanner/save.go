package scanner

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/tags"
	"gorm.io/gorm/clause"
)

type imageTask struct {
	path     string
	coverArt string
}

type saveContext struct {
	seenArtists         map[string]bool
	seenGenres          map[string]bool
	seenAlbumsWithCover map[string]bool
	imageTasks          chan imageTask
	cacheDir            string
}

func (s *Scanner) saveResults(resultChan <-chan scanResult, cacheDir string, numWorkers int) {
	ctx := &saveContext{
		seenArtists:         make(map[string]bool),
		seenGenres:          make(map[string]bool),
		seenAlbumsWithCover: make(map[string]bool),
		imageTasks:          make(chan imageTask, numWorkers*10),
		cacheDir:            cacheDir,
	}

	var imageWg sync.WaitGroup
	s.startImageWorkers(ctx, &imageWg, numWorkers)

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
			s.processMetadata(child, res.tags, res.path, ctx)
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
	close(ctx.imageTasks)
	imageWg.Wait()
}

func (s *Scanner) startImageWorkers(ctx *saveContext, wg *sync.WaitGroup, numWorkers int) {
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range ctx.imageTasks {
				if t.coverArt == "" {
					continue
				}
				p := filepath.Join(ctx.cacheDir, t.coverArt)
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

func (s *Scanner) processMetadata(child *models.Child, t *tags.Tags, path string, ctx *saveContext) {
	if t.Title != "" {
		child.Title = t.Title
	}
	if t.Artist != "" {
		child.Artist = t.Artist
		child.Artists = s.getArtistsFromNames(t.Artists, ctx.seenArtists)
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
		child.Genres = s.getGenresFromNames(t.Genres, ctx.seenGenres)
	}
	if t.Lyrics != "" {
		child.Lyrics = t.Lyrics
	}
	child.Duration = t.Duration
	child.BitRate = t.Bitrate

	// Album logic
	if child.Album != "" {
		s.handleAlbum(child, t, path, ctx)
	} else {
		// for child without album, use its own cover art
		child.CoverArt = child.ID
		ctx.imageTasks <- imageTask{path: path, coverArt: child.CoverArt}
	}
}

func (s *Scanner) handleAlbum(child *models.Child, t *tags.Tags, path string, ctx *saveContext) {
	albumArtistStr := t.AlbumArtist
	var albumArtists []models.ArtistID3
	if albumArtistStr != "" {
		albumArtists = s.getArtistsFromNames(t.AlbumArtists, ctx.seenArtists)
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

	if !ctx.seenAlbumsWithCover[albumID] {
		created := time.Now()
		if child.Created != nil {
			created = *child.Created
		}
		album := models.AlbumID3{
			ID:       albumID,
			Name:     child.Album,
			Artist:   displayArtist,
			Created:  created,
			CoverArt: "al-" + albumID,
		}
		if len(groupArtists) > 0 {
			album.ArtistID = groupArtists[0].ID
			album.Artists = groupArtists
		}

		s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&album)
		ctx.seenAlbumsWithCover[albumID] = true
	}
	// Queue an image task for every song in the album to ensure we get cover art from any song that has it
	ctx.imageTasks <- imageTask{path: path, coverArt: "al-" + albumID}
	child.CoverArt = "al-" + albumID
}

func (s *Scanner) getArtistsFromNames(names []string, seen map[string]bool) []models.ArtistID3 {
	var artists []models.ArtistID3
	for _, name := range names {
		artistID := GenerateArtistID(name)
		artist := models.ArtistID3{
			ID:       artistID,
			Name:     name,
			CoverArt: "ar-" + artistID,
		}
		if !seen[artistID] {
			s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&artist)
			seen[artistID] = true
		}
		artists = append(artists, artist)
	}
	return artists
}

func (s *Scanner) getGenresFromNames(names []string, seen map[string]bool) []models.Genre {
	var genres []models.Genre
	for _, name := range names {
		genre := models.Genre{Name: name}
		if !seen[name] {
			s.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&genre)
			seen[name] = true
		}
		genres = append(genres, genre)
	}
	return genres
}
