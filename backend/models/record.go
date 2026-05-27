package models

import "time"

type Record struct {
    ID            uint      `gorm:"primaryKey" json:"id"`
    UserID        uint      `gorm:"index;not null" json:"user_id"`
    User          User      `gorm:"foreignKey:UserID" json:"-"`
    SingleNumber  int       `json:"single_number"`  // e.g. 41（穩定識別鍵）
    SingleName    string    `json:"single_name"`    // e.g. "41stシングル「歌名」"（顯示用）
    LotteryRound  string    `json:"lottery_round"`  // e.g. "第3次"
    MemberName    string    `gorm:"not null" json:"member_name"`
    EventDate     string    `gorm:"not null" json:"event_date"`
    Session       string    `gorm:"not null" json:"session"`
    AppliedCount  int       `gorm:"not null" json:"applied_count"`
    WonCount      int       `gorm:"not null" json:"won_count"`
    SourceURL     string    `json:"source_url"`
    ScrapedAt     time.Time `json:"scraped_at"`
}
