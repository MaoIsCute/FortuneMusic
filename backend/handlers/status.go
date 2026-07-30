package handlers

import (
	"net/http"

	"fortune-tracker/db"
	"fortune-tracker/middleware"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

func loadSiteStatus() models.SiteStatus {
	var s models.SiteStatus
	db.DB.FirstOrCreate(&s, models.SiteStatus{ID: 1})
	return s
}

// GetStatus 是公開端點（無需登入），供 App.vue 啟動時檢查維護模式、擴充功能相關頁面
// 檢查目前最新版本號，取代原本讀 GitHub raw status.json 的做法。
func GetStatus(c *gin.Context) {
	s := loadSiteStatus()
	c.JSON(http.StatusOK, gin.H{
		"maintenance":              s.Maintenance,
		"message":                  s.Message,
		"latest_extension_version": middleware.LatestExtensionVersion,
	})
}

// UpdateStatus 讓管理者開關維護模式、修改提示訊息，管理者限定。
func UpdateStatus(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	var req struct {
		Maintenance bool   `json:"maintenance"`
		Message     string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 maintenance 與 message"})
		return
	}
	s := loadSiteStatus()
	db.DB.Model(&s).Updates(map[string]any{"maintenance": req.Maintenance, "message": req.Message})
	c.JSON(http.StatusOK, gin.H{"maintenance": req.Maintenance, "message": req.Message})
}
