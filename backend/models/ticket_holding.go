package models

import "time"

// TicketHolding 是 forTUNE meets 券面頁（fortunemusic.jp/mypage/online_handshake_detail/{id}/）
// 一列的快照：某張單曲某天某部某成員目前「實際持有」幾張（保有数）、用掉幾張（使用数）。
//
// 跟 records.won_count（当選数）的差別：保有数只算付過款的，放棄付款的中選不會出現在這裡，
// 而且已經把前幾抽付過款的張數加總好了——二抽付款模擬（/online-calc）拿它當「已確定」基準，
// 不用自己猜使用者付了哪幾張單。
//
// 這是快照不是歷史：擴充功能每次同步都送整張單曲的完整券面，後端先刪掉該使用者該單曲的
// 舊列再寫入（見 handlers.PushTicketHoldings），所以表裡永遠只有最近一次同步的狀態。
// 券面頁一張單曲只有一頁（不是一張抽選單一頁），用 (group, single_number) 對應單曲；
// 專輯沒有可靠編號（見 CLAUDE.md「單曲 vs 專輯」），目前擴充功能只送單曲。
type TicketHolding struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"index:idx_ticket_holding_single;not null" json:"user_id"`
	Group        string    `gorm:"index:idx_ticket_holding_single" json:"group"`
	SingleNumber int       `gorm:"index:idx_ticket_holding_single" json:"single_number"`
	SingleName   string    `json:"single_name"`
	EventDate    string    `json:"event_date"` // "YYYY/MM/DD"，跟 records 同格式才能對上同一格
	Session      string    `json:"session"`    // "第N部"
	TimeRange    string    `json:"time_range"` // 券面上的時刻，例 "12:00~13:30"（records 沒有這個資訊）
	MemberName   string    `json:"member_name"`
	HeldCount    int       `json:"held_count"` // 保有数
	UsedCount    int       `json:"used_count"` // 使用数
	ScrapedAt    time.Time `json:"scraped_at"`
}

// UnpaidApply 是「お支払い方法の未登録一覧」（payment_request_list）上還沒付款的中選單快照，
// 跟 TicketHolding 同一次同步一起送、一起整份覆蓋（每個使用者只保留最近一次的清單）。
// ApplySeq 是応募番号「76-4105-3448305-00」中間那段，對應 records.order_id（apply_detail id，
// 例 13448305）的尾碼——付款模擬用「order_id 結尾是 ApplySeq」判斷這張單還沒付款。
// 付過款的單已經算進券面保有数，不能再讓使用者勾，否則會重複計算。
type UnpaidApply struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	ApplySeq  string    `gorm:"not null" json:"apply_seq"`
	ScrapedAt time.Time `json:"scraped_at"`
}
