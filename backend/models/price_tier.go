package models

// PriceTier 記錄「從某張單曲開始，握手券單價變成多少」的級距設定（例：乃木坂46 從 34 單開始
// 是 2000 日圓、之前是 1900），不是每張單曲各自登記一筆——這樣之後新單曲上線只要還沒漲價，
// 不用每次都手動登記，自動沿用最新的級距。查詢時找「該團體裡 from_single_number 小於等於
// 目前單曲號、且最接近」的那一筆。
type PriceTier struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	Group            string `gorm:"uniqueIndex:idx_price_tier_group_from;not null" json:"group"`
	FromSingleNumber int    `gorm:"uniqueIndex:idx_price_tier_group_from;not null" json:"from_single_number"`
	UnitPrice        int    `gorm:"not null" json:"unit_price"`
}
