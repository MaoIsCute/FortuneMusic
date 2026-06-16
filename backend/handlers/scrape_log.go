package handlers

import (
	"net/http"
	"time"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

func PushScrapeLog(c *gin.Context) {
	var req struct {
		ScrapeToken string `json:"scrape_token" binding:"required"`
		Type        string `json:"type" binding:"required"`
		NewCount    int    `json:"new_count"`
		SkipCount   int    `json:"skip_count"`
		Error       string `json:"error"`
		DurationSec int    `json:"duration_sec"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "參數錯誤"})
		return
	}

	var user models.User
	if err := db.DB.Where("scrape_token = ?", req.ScrapeToken).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的 scrape token"})
		return
	}

	entry := models.ScrapeLog{
		UserID:      user.ID,
		Type:        req.Type,
		NewCount:    req.NewCount,
		SkipCount:   req.SkipCount,
		Error:       req.Error,
		DurationSec: req.DurationSec,
		CreatedAt:   time.Now(),
	}
	db.DB.Create(&entry)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func GetAdminScrapeLogs(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	type logRow struct {
		ID          uint      `json:"id"`
		UserName    string    `json:"user_name"`
		UserEmail   string    `json:"user_email"`
		Type        string    `json:"type"`
		NewCount    int       `json:"new_count"`
		SkipCount   int       `json:"skip_count"`
		Error       string    `json:"error"`
		DurationSec int       `json:"duration_sec"`
		CreatedAt   time.Time `json:"created_at"`
	}

	var rows []logRow
	db.DB.Model(&models.ScrapeLog{}).
		Select("scrape_logs.id, users.name as user_name, users.email as user_email, scrape_logs.type, scrape_logs.new_count, scrape_logs.skip_count, scrape_logs.error, scrape_logs.duration_sec, scrape_logs.created_at").
		Joins("JOIN users ON users.id = scrape_logs.user_id").
		Order("scrape_logs.created_at DESC").
		Limit(50).
		Scan(&rows)

	if rows == nil {
		rows = []logRow{}
	}
	c.JSON(http.StatusOK, rows)
}
