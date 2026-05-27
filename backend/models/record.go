package models

import "time"

type Record struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	User          User      `gorm:"foreignKey:UserID" json:"-"`
	OrderID       string    `gorm:"index" json:"order_id"`
	SingleNumber  int       `json:"single_number"`
	SingleName    string    `json:"single_name"`
	LotteryRound  string    `json:"lottery_round"`
	MemberName    string    `gorm:"not null" json:"member_name"`
	EventDate     string    `gorm:"not null" json:"event_date"`
	Session       string    `gorm:"not null" json:"session"`
	AppliedCount  int       `gorm:"not null" json:"applied_count"`
	WonCount      int       `gorm:"not null" json:"won_count"`
	SourceURL     string    `json:"source_url"`
	ScrapedAt     time.Time `json:"scraped_at"`
}
