package browser

import (
	"gorm.io/gorm"
)

type Browser struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Browser {
	return &Browser{db: db}
}
