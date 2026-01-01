package models

import (
	"encoding/json"
	"encoding/xml"
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
	CreatedAt        time.Time      `gorm:"index" json:"-" xml:"-"`
	UpdatedAt        time.Time      `json:"-" xml:"-"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-" xml:"-"`
	Username         string         `gorm:"primaryKey" xml:"username,attr" json:"username"`
	Password         string         `xml:"-" json:"-"`
	Email            string         `xml:"email,attr,omitempty" json:"email,omitempty"`
	AdminRole        bool           `xml:"adminRole,attr" json:"adminRole"`
	SubsonicSettings `gorm:"embedded"`
}

func (u *User) prepare() {
	if size := len(u.MusicFolders); size > 0 {
		u.FolderIDs = make([]uint, 0, size)
		for _, f := range u.MusicFolders {
			u.FolderIDs = append(u.FolderIDs, f.ID)
		}
	}
}

func (u *User) MarshalJSON() ([]byte, error) {
	u.prepare()
	type Alias User
	return json.Marshal((*Alias)(u))
}

func (u *User) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	u.prepare()
	type Alias User
	return e.EncodeElement((*Alias)(u), start)
}

type SubsonicSettings struct {
	ScrobblingEnabled   bool          `gorm:"default:true" xml:"scrobblingEnabled,attr" json:"scrobblingEnabled"`
	MaxBitRate          int           `xml:"maxBitRate,attr,omitempty" json:"maxBitRate,omitempty"`
	SettingsRole        bool          `xml:"settingsRole,attr" json:"settingsRole"`
	DownloadRole        bool          `gorm:"default:true" xml:"downloadRole,attr" json:"downloadRole"`
	UploadRole          bool          `xml:"uploadRole,attr" json:"uploadRole"`
	PlaylistRole        bool          `gorm:"default:true" xml:"playlistRole,attr" json:"playlistRole"`
	CoverArtRole        bool          `gorm:"default:true" xml:"coverArtRole,attr" json:"coverArtRole"`
	CommentRole         bool          `gorm:"default:true" xml:"commentRole,attr" json:"commentRole"`
	PodcastRole         bool          `xml:"podcastRole,attr" json:"podcastRole"`
	StreamRole          bool          `gorm:"default:true" xml:"streamRole,attr" json:"streamRole"`
	JukeboxRole         bool          `xml:"jukeboxRole,attr" json:"jukeboxRole"`
	ShareRole           bool          `gorm:"default:true" xml:"shareRole,attr" json:"shareRole"`
	VideoConversionRole bool          `xml:"videoConversionRole,attr" json:"videoConversionRole"`
	AvatarLastChanged   *time.Time    `xml:"avatarLastChanged,attr,omitempty" json:"avatarLastChanged,omitempty"`
	MusicFolders        []MusicFolder `gorm:"many2many:user_music_folders;" xml:"-" json:"-"`
	FolderIDs           []uint        `gorm:"-" xml:"folder,omitempty" json:"folder,omitempty"`
}
