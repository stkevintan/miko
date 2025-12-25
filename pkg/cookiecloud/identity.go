package cookiecloud

import (
	"time"

	"gorm.io/gorm"
)

type Identity struct {
	Username  string         `gorm:"primaryKey" json:"username"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UUID      string         `gorm:"type:text" json:"uuid"`
	Password  string         `gorm:"type:text" json:"password"`
}

func (Identity) TableName() string {
	return "cookiecloud_identities"
}
