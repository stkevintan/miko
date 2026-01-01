package scanner

import (
	"context"
	"sync"

	"github.com/stkevintan/miko/pkg/log"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (s *Scanner) Prune(ctx context.Context, seenIDs *sync.Map) {
	log.Info("Pruning deleted files and orphaned records...")

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Create a temporary table to store seen IDs
		tx.Exec("CREATE TEMPORARY TABLE seen_ids (id TEXT PRIMARY KEY)")
		defer tx.Exec("DROP TABLE seen_ids")

		// 2. Insert all seen IDs into the temporary table
		var ids []string
		seenIDs.Range(func(key, value any) bool {
			ids = append(ids, key.(string))
			if len(ids) >= 500 {
				s.insertSeenIDs(tx, ids)
				ids = ids[:0]
			}
			return true
		})
		if len(ids) > 0 {
			s.insertSeenIDs(tx, ids)
		}

		// 3. Delete children that are NOT in the seen_ids table
		result := tx.Exec("DELETE FROM children WHERE NOT EXISTS (SELECT 1 FROM seen_ids WHERE seen_ids.id = children.id)")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			log.Info("Pruned %d deleted files from database", result.RowsAffected)
		}

		// 4. Prune orphaned albums (albums with no songs)
		result = tx.Exec(`
			DELETE FROM album_id3 
			WHERE NOT EXISTS (SELECT 1 FROM children WHERE children.album_id = album_id3.id)
		`)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			log.Info("Pruned %d orphaned albums", result.RowsAffected)
		}

		// 5. Prune orphaned join table entries
		// We must do this BEFORE pruning artists and genres because they rely on these tables
		if err := tx.Exec(`DELETE FROM song_artists WHERE child_id NOT IN (SELECT id FROM children)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM album_artists WHERE album_id3_id NOT IN (SELECT id FROM album_id3)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM song_genres WHERE child_id NOT IN (SELECT id FROM children)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM album_genres WHERE album_id3_id NOT IN (SELECT id FROM album_id3)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM playlist_songs WHERE song_id NOT IN (SELECT id FROM children)`).Error; err != nil {
			return err
		}

		// 6. Prune orphaned artists
		// This is a bit more complex because artists can be linked to songs or albums
		result = tx.Exec(`
			DELETE FROM artist_id3 
			WHERE NOT EXISTS (SELECT 1 FROM children WHERE children.artist_id = artist_id3.id)
			AND NOT EXISTS (SELECT 1 FROM album_id3 WHERE album_id3.artist_id = artist_id3.id)
			AND NOT EXISTS (SELECT 1 FROM song_artists WHERE song_artists.artist_id3_id = artist_id3.id)
			AND NOT EXISTS (SELECT 1 FROM album_artists WHERE album_artists.artist_id3_id = artist_id3.id)
		`)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			log.Info("Pruned %d orphaned artists", result.RowsAffected)
		}

		// 7. Prune orphaned genres
		result = tx.Exec(`
			DELETE FROM genres 
			WHERE NOT EXISTS (SELECT 1 FROM song_genres WHERE song_genres.genre_name = genres.name)
			AND NOT EXISTS (SELECT 1 FROM album_genres WHERE album_genres.genre_name = genres.name)
		`)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			log.Info("Pruned %d orphaned genres", result.RowsAffected)
		}

		return nil
	})

	if err != nil {
		log.Error("Failed to prune database: %v", err)
	}
}

func (s *Scanner) insertSeenIDs(db *gorm.DB, ids []string) {
	if len(ids) == 0 {
		return
	}
	data := make([]map[string]any, len(ids))
	for i, id := range ids {
		data[i] = map[string]any{"id": id}
	}
	// GORM will automatically use a single multi-row INSERT statement for better performance.
	// Clauses(clause.OnConflict{DoNothing: true}) is equivalent to INSERT OR IGNORE.
	if err := db.Table("seen_ids").Clauses(clause.OnConflict{DoNothing: true}).Create(data).Error; err != nil {
		log.Error("Failed to insert seen IDs: %v", err)
	}
}
