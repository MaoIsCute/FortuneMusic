package models

import "gorm.io/gorm"

type TitleCorrection struct {
	gorm.Model
	SingleNumber int    `gorm:"uniqueIndex;not null"`
	SingleName   string `gorm:"not null"`
}
