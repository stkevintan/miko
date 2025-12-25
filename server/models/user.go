package models

import (
	"time"

	"gorm.io/gorm"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type User struct {
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Username  string         `gorm:"primaryKey" xml:"username,attr" json:"username"`
	Password  string         `xml:"password,attr" json:"password"`
	Email     string         `xml:"email,attr,omitempty" json:"email,omitempty"`
	IsAdmin   bool           `json:"is_admin"`
	// ScrobblingEnabled   bool       `xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	// MaxBitRate          int        `xml:"maxBitRate,attr,omitempty" json:"maxBitRate,omitempty"`
	// AdminRole           bool       `xml:"adminRole,attr" json:"adminRole"`
	// SettingsRole        bool       `xml:"settingsRole,attr" json:"settingsRole"`
	// DownloadRole        bool       `xml:"downloadRole,attr" json:"downloadRole"`
	// UploadRole          bool       `xml:"uploadRole,attr" json:"uploadRole"`
	// PlaylistRole        bool       `xml:"playlistRole,attr" json:"playlistRole"`
	// CoverArtRole        bool       `xml:"coverArtRole,attr" json:"coverArtRole"`
	// CommentRole         bool       `xml:"commentRole,attr" json:"commentRole"`
	// PodcastRole         bool       `xml:"podcastRole,attr" json:"podcastRole"`
	// StreamRole          bool       `xml:"streamRole,attr" json:"streamRole"`
	// JukeboxRole         bool       `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	// ShareRole           bool       `xml:"shareRole,attr" json:"shareRole"`
	// VideoConversionRole bool       `xml:"videoConversionRole,attr" json:"videoConversionRole"`
	// AvatarLastChanged   *time.Time `xml:"avatarLastChanged,attr,omitempty" json:"avatarLastChanged,omitempty"`
	// Folder              []int      `xml:"folder,omitempty" json:"folder,omitempty"`
}
