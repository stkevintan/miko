package models

import (
	"time"
)

type PlaylistRecord struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"index" json:"name"`
	Comment   string         `json:"comment"`
	Owner     string         `gorm:"index" json:"owner"`
	Public    bool           `json:"public"`
	Songs     []PlaylistSong `gorm:"foreignKey:PlaylistID;constraint:OnDelete:CASCADE" json:"songs"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type PlaylistSong struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	PlaylistID uint   `gorm:"index" json:"playlistId"`
	SongID     string `gorm:"index" json:"songId"`
	Position   int    `json:"position"`
}
