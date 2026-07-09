package models

import (
	"time"

	"gorm.io/gorm"
)

type Title struct {
	gorm.Model
	Group        string     `gorm:"uniqueIndex:idx_title_group_single;not null"`
	SingleNumber int        `gorm:"uniqueIndex:idx_title_group_single;not null"`
	OrgAlbumName string     `gorm:"uniqueIndex:idx_title_group_single;not null;default:''"`
	SingleName   string     `gorm:"not null"`
	ReleaseDate  *time.Time `gorm:"type:date"`
}
