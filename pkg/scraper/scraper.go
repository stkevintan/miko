package scraper

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/stkevintan/miko/config"
	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
	"github.com/stkevintan/miko/pkg/musicbrainz"
	"github.com/stkevintan/miko/pkg/scanner"
	"github.com/stkevintan/miko/pkg/shared"
	"github.com/stkevintan/miko/pkg/tags"
	"gorm.io/gorm"
)

type Scraper struct {
	walker     *shared.Walker
	mb         *musicbrainz.Client
	db         *gorm.DB
	isScraping atomic.Bool
	scanner    *scanner.Scanner
	cfg        *config.Config
}

func New(db *gorm.DB, cfg *config.Config, s *scanner.Scanner) *Scraper {
	return &Scraper{
		db:      db,
		walker:  shared.NewWalker(db, cfg),
		mb:      musicbrainz.NewClient(),
		scanner: s,
		cfg:     cfg,
	}
}

func (c *Scraper) IsScraping() bool {
	return c.isScraping.Load()
}

func (c *Scraper) ScrapePath(ctx context.Context, id string, mode string) (*sync.Map, error) {
	taskChan, err := c.walker.WalkByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return c.scrape(ctx, taskChan, mode)
}

func (c *Scraper) ScrapeAll(ctx context.Context, mode string) (*sync.Map, error) {
	taskChan, err := c.walker.WalkAllRoots(ctx)
	if err != nil {
		return nil, err
	}

	return c.scrape(ctx, taskChan, mode)
}

func (c *Scraper) scrape(ctx context.Context, taskChan <-chan shared.WalkTask, mode string) (*sync.Map, error) {
	if !c.isScraping.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("scrape already in progress")
	}
	defer c.isScraping.Store(false)

	if mode == "" {
		mode = c.cfg.Subsonic.ScrapeMode
	}

	var tasks []shared.WalkTask
	for task := range taskChan {
		tasks = append(tasks, task)
		if task.D.IsDir() {
			continue
		}
		if !shared.IsAudioFile(task.Path) {
			continue
		}

		// Look up song in DB to get current metadata for search
		var song models.Child
		if err := c.db.Where("path = ?", task.Path).First(&song).Error; err != nil {
			continue
		}

		// Incremental scrape: skip if already has MusicBrainz Track ID
		if mode == "inc" {
			if currentTags, err := tags.ReadAll(song.Path); err == nil {
				if trackIDs, ok := currentTags[tags.MusicBrainzTrackID]; ok && len(trackIDs) > 0 && trackIDs[0] != "" {
					log.Info("[Inc mode] Skipping song with existing MusicBrainz Track ID: %s", song.Path)
					continue
				}
			}
		}

		log.Info("Scraping metadata for song: %s", song.Path)

		if err := c.ScrapeSong(ctx, &song); err != nil {
			log.Error("Failed to scrape song %s: %v", song.Path, err)
			continue
		} else {
			log.Info("Successfully scraped song: %s", song.Path)
		}
	}

	// Create a new channel to feed the collected tasks to the scanner
	scanChan := make(chan shared.WalkTask, len(tasks))
	for _, t := range tasks {
		scanChan <- t
	}
	close(scanChan)

	// after scraping, run a scan to update the database with new tags
	return c.scanner.Scan(ctx, false, scanChan)
}

func (c *Scraper) ScrapeSong(ctx context.Context, song *models.Child) error {
	searchResult, err := c.mb.SearchRecording(ctx, song.Artist, song.Album, song.Title)
	if err != nil {
		return fmt.Errorf("failed to search on MusicBrainz: %w", err)
	}

	// Get full details
	recording, err := c.mb.GetRecording(ctx, searchResult.ID)
	if err != nil {
		// If fetching full details fails, we cannot proceed with enriching the tags
		// as the searchResult is only a summary.
		return fmt.Errorf("failed to get full recording details from MusicBrainz: %w", err)
	}

	// Prepare tags for update
	newTags := map[string][]string{
		tags.Title:              {recording.Title},
		tags.MusicBrainzTrackID: {recording.ID},
	}

	if len(recording.Artists) > 0 {
		artists := make([]string, len(recording.Artists))
		artistIDs := make([]string, len(recording.Artists))
		for i, a := range recording.Artists {
			artists[i] = a.Name
			artistIDs[i] = a.ID
		}
		newTags[tags.Artists] = artists
		newTags[tags.MusicBrainzArtistID] = artistIDs
	}

	if len(recording.ISRCs) > 0 {
		newTags[tags.ISRC] = recording.ISRCs
	}

	// Genres and Tags
	genres := make([]string, 0)
	for _, g := range recording.Genres {
		genres = append(genres, g.Name)
	}
	for _, t := range recording.Tags {
		genres = append(genres, t.Name)
	}
	if len(genres) > 0 {
		newTags[tags.Genre] = genres
	}

	// Relationships (Composer, Lyricist, etc.)
	addUniqueTag := func(key string, value string) {
		if value == "" {
			return
		}
		for _, v := range newTags[key] {
			if v == value {
				return
			}
		}
		newTags[key] = append(newTags[key], value)
	}

	for _, rel := range recording.Relations {
		switch rel.Type {
		case "composer":
			addUniqueTag(tags.Composer, rel.Artist.Name)
		case "lyricist":
			addUniqueTag(tags.Lyricist, rel.Artist.Name)
		case "producer":
			addUniqueTag(tags.Producer, rel.Artist.Name)
		}

		// Check work relations for composer/lyricist if not found on recording
		if rel.Work.Title != "" {
			for _, wrel := range rel.Work.Relations {
				switch wrel.Type {
				case "composer":
					addUniqueTag(tags.Composer, wrel.Artist.Name)
				case "lyricist":
					addUniqueTag(tags.Lyricist, wrel.Artist.Name)
				}
			}
		}
	}

	if len(recording.Releases) > 0 {
		release := recording.Releases[0]
		// Try to find a release that matches the album name
		if song.Album != "" {
			for _, r := range recording.Releases {
				if strings.EqualFold(r.Title, song.Album) {
					release = r
					break
				}
			}
		}

		newTags[tags.Album] = []string{release.Title}
		newTags[tags.MusicBrainzAlbumID] = []string{release.ID}
		newTags[tags.MusicBrainzReleaseGroupID] = []string{release.ReleaseGroup.ID}

		if release.Status != "" {
			newTags[tags.ReleaseStatus] = []string{release.Status}
		}
		if release.Country != "" {
			newTags[tags.ReleaseCountry] = []string{release.Country}
		}
		if release.Barcode != "" {
			newTags[tags.Barcode] = []string{release.Barcode}
		}
		if release.ReleaseGroup.Type != "" {
			newTags[tags.ReleaseType] = []string{release.ReleaseGroup.Type}
		}

		if len(release.ArtistCredit) > 0 {
			albumArtists := make([]string, len(release.ArtistCredit))
			albumArtistIDs := make([]string, len(release.ArtistCredit))
			for i, a := range release.ArtistCredit {
				albumArtists[i] = a.Name
				albumArtistIDs[i] = a.ID
			}
			newTags[tags.AlbumArtist] = albumArtists
			newTags[tags.MusicBrainzAlbumArtistID] = albumArtistIDs
		}

		if release.Date != "" {
			newTags[tags.Date] = []string{release.Date}
		}

		if len(release.Media) > 0 {
			media := release.Media[0]
			newTags[tags.DiscNumber] = []string{fmt.Sprintf("%d", media.Position)}
			if media.Format != "" {
				newTags[tags.Media] = []string{media.Format}
			}
			if len(media.Track) > 0 {
				newTags[tags.TrackNumber] = []string{media.Track[0].Number}
			}
		}

		// Fetch cover art from Cover Art Archive
		log.Debug("Fetching cover art for release: %s", release.ID)
		if coverData, err := c.mb.FetchCoverArt(ctx, release.ID); err == nil {
			if err := tags.WriteImage(song.Path, coverData); err != nil {
				log.Warn("Failed to write cover art to %s: %v", song.Path, err)
			} else {
				log.Info("Successfully scraped cover art for: %s", song.Path)
			}
		} else {
			log.Info("No cover art found for release %s: %v", release.ID, err)
		}
	}

	// Update tags in file
	if err := tags.Write(song.Path, newTags); err != nil {
		return fmt.Errorf("failed to write tags to file: %w", err)
	}

	return nil
}
