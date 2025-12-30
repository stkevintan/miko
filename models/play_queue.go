package models

import "time"

type PlayQueueRecord struct {
	Username  string    `gorm:"primaryKey"`
	Current   string    `json:"current"`
	Position  int64     `json:"position"`
	Changed   time.Time `json:"changed"`
	ChangedBy string    `json:"changedBy"`
}

type PlayQueueSong struct {
	Username string `gorm:"primaryKey"`
	SongID   string `gorm:"primaryKey"`
	Position int    `gorm:"primaryKey"`
}
