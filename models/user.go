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
	CreatedAt        time.Time         `json:"-"`
	UpdatedAt        time.Time         `json:"-"`
	DeletedAt        gorm.DeletedAt    `gorm:"index" json:"-"`
	Username         string            `gorm:"primaryKey" xml:"username,attr" json:"username"`
	Password         string            `xml:"password,attr" json:"password"`
	Email            string            `xml:"email,attr,omitempty" json:"email,omitempty"`
	IsAdmin          bool              `json:"is_admin"`
	SubsonicSettings *SubsonicSettings `gorm:"foreignKey:Username" xml:",inline" json:"subsonic_settings,omitempty"`
}

func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	settings := SubsonicSettings{Username: u.Username}
	return tx.Create(&settings).Error
}

func (u *User) AfterFind(tx *gorm.DB) (err error) {
	if u.SubsonicSettings == nil {
		u.SubsonicSettings = &SubsonicSettings{
			ScrobblingEnabled: true,
			StreamRole:        true,
			DownloadRole:      true,
			PlaylistRole:      true,
			CoverArtRole:      true,
			CommentRole:       true,
			ShareRole:         true,
		}
	}
	return nil
}

type SubsonicSettings struct {
	Username            string     `gorm:"primaryKey" json:"-"`
	ScrobblingEnabled   bool       `gorm:"default:true" xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	MaxBitRate          int        `xml:"maxBitRate,attr,omitempty" json:"maxBitRate,omitempty"`
	AdminRole           bool       `xml:"adminRole,attr" json:"adminRole"`
	SettingsRole        bool       `xml:"settingsRole,attr" json:"settingsRole"`
	DownloadRole        bool       `gorm:"default:true" xml:"downloadRole,attr" json:"downloadRole"`
	UploadRole          bool       `xml:"uploadRole,attr" json:"uploadRole"`
	PlaylistRole        bool       `gorm:"default:true" xml:"playlistRole,attr" json:"playlistRole"`
	CoverArtRole        bool       `gorm:"default:true" xml:"coverArtRole,attr" json:"coverArtRole"`
	CommentRole         bool       `gorm:"default:true" xml:"commentRole,attr" json:"commentRole"`
	PodcastRole         bool       `xml:"podcastRole,attr" json:"podcastRole"`
	StreamRole          bool       `gorm:"default:true" xml:"streamRole,attr" json:"streamRole"`
	JukeboxRole         bool       `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	ShareRole           bool       `gorm:"default:true" xml:"shareRole,attr" json:"shareRole"`
	VideoConversionRole bool       `xml:"videoConversionRole,attr" json:"videoConversionRole"`
	AvatarLastChanged   *time.Time `xml:"avatarLastChanged,attr,omitempty" json:"avatarLastChanged,omitempty"`
}
