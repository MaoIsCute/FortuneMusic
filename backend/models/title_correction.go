package models

import "gorm.io/gorm"

type TitleCorrection struct {
	gorm.Model
	Group        string `gorm:"uniqueIndex:idx_title_correction_group_single;not null"`
	SingleNumber int    `gorm:"uniqueIndex:idx_title_correction_group_single;not null"`
	SingleName   string `gorm:"not null"`
}
