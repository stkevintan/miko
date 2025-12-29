package scanner

import (
	"os"
	"path/filepath"
	"time"

	"github.com/stkevintan/miko/models"
	"github.com/stkevintan/miko/pkg/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Scanner) saveResults(db *gorm.DB, resultChan <-chan scanResult, cacheDir string) {
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

	for res := range resultChan {
		child := res.child
		t := res.tags

		if t != nil {
			if t.Title != "" {
				child.Title = t.Title
			}
			if t.Artist != "" {
				child.Artist = t.Artist
				child.Artists = s.getArtistsFromNames(db, t.Artists, seenArtists)
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
				child.Genres = s.getGenresFromNames(db, t.Genres, seenGenres)
			}
			if t.Lyrics != "" {
				child.Lyrics = t.Lyrics
			}
			child.Duration = t.Duration
			child.BitRate = t.Bitrate

			// Album logic
			if child.Album != "" {
				albumArtistStr := t.AlbumArtist
				var albumArtists []models.ArtistID3
				if albumArtistStr != "" {
					albumArtists = s.getArtistsFromNames(db, t.AlbumArtists, seenArtists)
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

				if !seenAlbums[albumID] {
					created := time.Now()
					if child.Created != nil {
						created = *child.Created
					}
					album := models.AlbumID3{
						ID:      albumID,
						Name:    child.Album,
						Artist:  displayArtist,
						Created: created,
					}
					if len(groupArtists) > 0 {
						album.ArtistID = groupArtists[0].ID
						album.Artists = groupArtists
					}
					if len(t.Image) > 0 {
						album.CoverArt = album.ID
						if err := os.WriteFile(filepath.Join(cacheDir, album.ID), t.Image, 0644); err != nil {
							log.Warn("Failed to write album cover to cache for album %s: %v", album.ID, err)
						}
					}
					db.Clauses(clause.OnConflict{UpdateAll: true}).Create(&album)
					seenAlbums[albumID] = true
				}

				if _, err := os.Stat(filepath.Join(cacheDir, albumID)); err == nil {
					child.CoverArt = albumID
				}
			}

			if child.CoverArt == "" && len(t.Image) > 0 {
				child.CoverArt = child.ID
				if err := os.WriteFile(filepath.Join(cacheDir, child.ID), t.Image, 0644); err != nil {
					log.Warn("Failed to write song cover to cache for song %s: %v", child.ID, err)
				}
			}
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
}

func (s *Scanner) getArtistsFromNames(db *gorm.DB, names []string, seen map[string]bool) []models.ArtistID3 {
	var artists []models.ArtistID3
	for _, name := range names {
		artistID := GenerateArtistID(name)
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

func (s *Scanner) getGenresFromNames(db *gorm.DB, names []string, seen map[string]bool) []models.Genre {
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
