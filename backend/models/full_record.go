package models

import "time"

type FullRecord struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	UserID       uint   `gorm:"index;not null" json:"user_id"`
	User         User   `gorm:"foreignKey:UserID" json:"-"`
	OrderID      string `gorm:"index" json:"order_id"`
	Group        string `json:"group"`
	SingleNumber int    `json:"single_number"`
	SingleName   string `json:"single_name"`
	EventType    string `gorm:"not null" json:"event_type"` // 実体 / 線上
	Venue        string `json:"venue"`                      // 東京 / 地方 (実体 only)
	EventDate    string `gorm:"not null" json:"event_date"`
	Session      string `gorm:"not null" json:"session"`
	MemberName   string `gorm:"not null" json:"member_name"`
	AppliedCount int    `gorm:"not null" json:"applied_count"`
	WonCount     int    `gorm:"not null" json:"won_count"`
	// Pending 是這場抽選還沒開獎（來源網站 resultInfo 是 null）。true 時 WonCount 只是暫時的
	// 0，不代表真的落選，統計端點的中選率計算要排除 Pending 的記錄，不然會被當成確定落選拉低
	// 分母；等使用者之後重新同步、resultInfo 有值了才會轉回 false 並補上真正的 WonCount。
	Pending      bool      `gorm:"default:false" json:"pending"`
	LotteryRound float64   `json:"lottery_round"`
	SourceURL    string    `json:"source_url"`
	ScrapedAt    time.Time `json:"scraped_at"`
}
