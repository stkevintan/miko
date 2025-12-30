package bookmarks

import (
	"time"

	"github.com/stkevintan/miko/models"
	"gorm.io/gorm"
)

type Manager struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Manager {
	return &Manager{db: db}
}

func (m *Manager) GetBookmarks(username string) ([]models.Bookmark, error) {
	var results []struct {
		models.Child
		BPosition  int64     `gorm:"column:b_position"`
		BComment   string    `gorm:"column:b_comment"`
		BCreatedAt time.Time `gorm:"column:b_created_at"`
		BUpdatedAt time.Time `gorm:"column:b_updated_at"`
	}

	err := m.db.Table("children").
		Select("children.*, bookmark_records.position as b_position, bookmark_records.comment as b_comment, bookmark_records.created_at as b_created_at, bookmark_records.updated_at as b_updated_at").
		Joins("JOIN bookmark_records ON bookmark_records.song_id = children.id").
		Where("bookmark_records.username = ?", username).
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	bookmarks := make([]models.Bookmark, len(results))
	for i, r := range results {
		r.Child.BookmarkPosition = r.BPosition
		bookmarks[i] = models.Bookmark{
			Position: r.BPosition,
			Username: username,
			Comment:  r.BComment,
			Created:  r.BCreatedAt,
			Changed:  r.BUpdatedAt,
			Entry:    r.Child,
		}
	}

	return bookmarks, nil
}

func (m *Manager) CreateBookmark(username, songID string, position int64, comment string) error {
	record := models.BookmarkRecord{
		Username: username,
		SongID:   songID,
		Position: position,
		Comment:  comment,
	}
	return m.db.Save(&record).Error
}

func (m *Manager) DeleteBookmark(username, songID string) error {
	return m.db.Where("username = ? AND song_id = ?", username, songID).Delete(&models.BookmarkRecord{}).Error
}

func (m *Manager) GetPlayQueue(username string) (*models.PlayQueue, error) {
	var record models.PlayQueueRecord
	if err := m.db.Where("username = ?", username).First(&record).Error; err != nil {
		return nil, err
	}

	var songs []models.Child
	err := m.db.Table("children").
		Joins("JOIN play_queue_songs ON play_queue_songs.song_id = children.id").
		Where("play_queue_songs.username = ?", username).
		Order("play_queue_songs.position ASC").
		Find(&songs).Error
	if err != nil {
		return nil, err
	}

	return &models.PlayQueue{
		Current:   record.Current,
		Position:  record.Position,
		Username:  record.Username,
		Changed:   record.Changed,
		ChangedBy: record.ChangedBy,
		Entry:     songs,
	}, nil
}

func (m *Manager) SavePlayQueue(username string, current string, position int64, songIDs []string, clientName string) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		record := models.PlayQueueRecord{
			Username:  username,
			Current:   current,
			Position:  position,
			Changed:   time.Now(),
			ChangedBy: clientName,
		}
		if err := tx.Save(&record).Error; err != nil {
			return err
		}

		if err := tx.Where("username = ?", username).Delete(&models.PlayQueueSong{}).Error; err != nil {
			return err
		}

		if len(songIDs) > 0 {
			queueSongs := make([]models.PlayQueueSong, len(songIDs))
			for i, songID := range songIDs {
				queueSongs[i] = models.PlayQueueSong{
					Username: username,
					SongID:   songID,
					Position: i,
				}
			}
			if err := tx.Create(&queueSongs).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
