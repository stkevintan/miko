package cookiecloud

import (
	"time"

	"gorm.io/gorm"
)

type Identity struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Username  string         `gorm:"uniqueIndex" json:"username"`
	UUID      string         `gorm:"type:text" json:"uuid"`
	Password  string         `gorm:"type:text" json:"password"`
	URL       string         `gorm:"type:text" json:"url"`
}

func (Identity) TableName() string {
	return "cookiecloud_identities"
}
