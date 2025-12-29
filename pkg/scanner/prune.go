package scanner

import (
	"sync"

	"github.com/stkevintan/miko/pkg/log"
	"gorm.io/gorm"
)

func (s *Scanner) Prune(db *gorm.DB, seenIDs *sync.Map) {
	log.Info("Pruning deleted files and orphaned records...")

	// 1. Create a temporary table to store seen IDs
	db.Exec("CREATE TEMPORARY TABLE seen_ids (id TEXT PRIMARY KEY)")
	defer db.Exec("DROP TABLE seen_ids")

	// 2. Insert all seen IDs into the temporary table
	// We do this in batches to be efficient
	var ids []string
	seenIDs.Range(func(key, value any) bool {
		ids = append(ids, key.(string))
		if len(ids) >= 500 {
			s.insertSeenIDs(db, ids)
			ids = ids[:0]
		}
		return true
	})
	if len(ids) > 0 {
		s.insertSeenIDs(db, ids)
	}

	// 3. Delete children that are NOT in the seen_ids table
	result := db.Exec("DELETE FROM children WHERE id NOT IN (SELECT id FROM seen_ids)")
	if result.Error != nil {
		log.Error("Failed to prune deleted files: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Info("Pruned %d deleted files from database", result.RowsAffected)
	}

	// 4. Prune orphaned albums (albums with no songs)
	result = db.Exec("DELETE FROM album_id3 WHERE id NOT IN (SELECT DISTINCT album_id FROM children WHERE album_id IS NOT NULL AND album_id != '')")
	if result.Error != nil {
		log.Error("Failed to prune orphaned albums: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Info("Pruned %d orphaned albums", result.RowsAffected)
	}

	// 5. Prune orphaned artists (artists with no songs and no albums)
	// This is a bit more complex because artists can be linked to songs or albums
	result = db.Exec(`
		DELETE FROM artist_id3 
		WHERE id NOT IN (SELECT DISTINCT artist_id FROM children WHERE artist_id IS NOT NULL AND artist_id != '')
		AND id NOT IN (SELECT DISTINCT artist_id FROM album_id3 WHERE artist_id IS NOT NULL AND artist_id != '')
		AND id NOT IN (SELECT DISTINCT artist_id3_id FROM artist_id3_songs)
		AND id NOT IN (SELECT DISTINCT artist_id3_id FROM artist_id3_albums)
	`)
	if result.Error != nil {
		log.Error("Failed to prune orphaned artists: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Info("Pruned %d orphaned artists", result.RowsAffected)
	}
}

func (s *Scanner) insertSeenIDs(db *gorm.DB, ids []string) {
	if len(ids) == 0 {
		return
	}
	// SQLite doesn't support multiple rows in a single INSERT easily with raw SQL without some tricks,
	// but GORM's CreateInBatches works on models. Since we don't have a model for the temp table,
	// we'll just use a simple loop or a prepared statement.
	tx := db.Begin()
	sqlDB, _ := tx.DB()
	stmt, _ := sqlDB.Prepare("INSERT OR IGNORE INTO seen_ids (id) VALUES (?)")
	for _, id := range ids {
		stmt.Exec(id)
	}
	stmt.Close()
	tx.Commit()
}
