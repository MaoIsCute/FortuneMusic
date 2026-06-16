package models

import "time"

type ScrapeLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
	Type      string    `gorm:"not null" json:"type"`
	NewCount    int    `json:"new_count"`
	SkipCount   int    `json:"skip_count"`
	Error       string `json:"error"`
	DurationSec int    `json:"duration_sec"`
	CreatedAt time.Time `json:"created_at"`
}
