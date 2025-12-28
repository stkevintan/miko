package models

import "gorm.io/gorm"

type SystemSetting struct {
	gorm.Model
	Key   string `gorm:"uniqueIndex"`
	Value string
}
