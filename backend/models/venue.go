package models

import "gorm.io/gorm"

// Venue 記錄實體場次「(group, single_number, event_date) → 場地」的人工登記對照，
// 補足早期抓取版本沒有解析場地欄位、之後也無法從來源網站回溯的舊資料缺口，
// 同時供新資料匯入時自動套用（同一張單同一天的場次只會在同一個場地舉辦）。
type Venue struct {
	gorm.Model
	Group        string `gorm:"uniqueIndex:idx_venue_group_single_date;not null"`
	SingleNumber int    `gorm:"uniqueIndex:idx_venue_group_single_date;not null"`
	EventDate    string `gorm:"uniqueIndex:idx_venue_group_single_date;not null"`
	VenueName    string `gorm:"not null"`
}
