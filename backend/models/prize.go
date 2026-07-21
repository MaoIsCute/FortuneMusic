package models

import "time"

// Prize 記錄「商品抽選」類型的獎品申請（生寫、海報等），跟握手/簽名會是不同性質的資料——
// history2 API 對這類獎品永遠回傳 result: "抽選中"，讀不到中選/落選，只有申請口數有意義，
// 所以不像 FullRecord/SignEvent 那樣記 won_count，也沒有 event_date/venue/session。
// PrizeCode 存來源網站原始值（如 "p_sign_photo"），中文名稱只在前端顯示層轉換，理由跟
// titles 表存日文原文一致（CLAUDE.md #110）：以後網站增加新獎品或顯示名稱要調整時，
// 不用回頭改資料庫裡已存的值。
type Prize struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	UserID       uint   `gorm:"index;not null" json:"user_id"`
	Group        string `json:"group"`
	SingleNumber int    `json:"single_number"`
	PrizeCode    string `gorm:"not null" json:"prize_code"`
	MemberName   string `gorm:"not null" json:"member_name"`
	AppliedCount int    `gorm:"not null" json:"applied_count"`
	// WonStatus 是使用者自己手動標記的中選結果（""=抽選中/未知、"won"=中選、"lost"=落選）——
	// 來源網站對這類獎品讀不到中選結果，只能讓使用者自己記錄。重新同步（pushPrizes）只會
	// 更新 AppliedCount，不會動這個欄位，避免使用者標記過的結果被下次同步洗掉。
	WonStatus string    `gorm:"default:''" json:"won_status"`
	ScrapedAt time.Time `json:"scraped_at"`
}
