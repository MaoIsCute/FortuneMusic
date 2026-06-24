package models

import "time"

type Purchase struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"index;not null" json:"user_id"`
	User         User       `gorm:"foreignKey:UserID" json:"-"`
	ItemKey      string     `gorm:"uniqueIndex;not null" json:"item_key"`
	EntryID      string     `gorm:"index;not null" json:"entry_id"`
	Group        string     `json:"group"`
	OrderNumber  string     `json:"order_number"`
	MemberName   string     `gorm:"not null" json:"member_name"`
	EventDate    string     `gorm:"not null" json:"event_date"`
	Session      string     `gorm:"not null" json:"session"`
	SingleNumber int        `json:"single_number"`
	SingleName   string     `json:"single_name"`
	LotteryRound int        `json:"lottery_round"`
	UnitPrice    int        `gorm:"not null" json:"unit_price"`
	Quantity     int        `gorm:"not null" json:"quantity"`
	Subtotal     int        `gorm:"not null" json:"subtotal"`
	AppliedAt    *time.Time `json:"applied_at"`
	ScrapedAt    time.Time  `json:"scraped_at"`
}
