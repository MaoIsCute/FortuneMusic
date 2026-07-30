package models

// SiteStatus 是網站維護模式的開關設定，全表固定只有一列（id=1）。透過
// PUT /api/admin/status 修改，改完立即生效不需要重新部署——取代原本手動編輯
// GitHub 上 status.json 檔案的做法（repo 改成 private 之後，前端直接打
// raw.githubusercontent.com 的公開網址會失效）。
type SiteStatus struct {
	ID          uint   `gorm:"primaryKey" json:"id"`
	Maintenance bool   `json:"maintenance"`
	Message     string `json:"message"`
}
