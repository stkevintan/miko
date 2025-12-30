package models

import "time"

type BookmarkRecord struct {
	Username  string    `gorm:"primaryKey"`
	SongID    string    `gorm:"primaryKey"`
	Position  int64     `json:"position"` // in milliseconds
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
