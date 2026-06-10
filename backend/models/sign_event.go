package models

import "time"

type SignEvent struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	OrderID      string    `gorm:"index" json:"order_id"`
	SingleNumber int       `json:"single_number"`
	SingleName   string    `json:"single_name"`
	EventDate    string    `json:"event_date"`
	MemberName   string    `json:"member_name"`
	AppliedCount int       `json:"applied_count"`
	WonCount     int       `json:"won_count"`
	LotteryRound float64   `json:"lottery_round"`
	ScrapedAt    time.Time `json:"scraped_at"`
}
