package handlers

import (
	"net/http"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

func GetRecords(c *gin.Context) {
	userID, _ := c.Get("userID")
	var records []models.Record
	db.DB.Where("user_id = ?", userID.(uint)).Order("scraped_at desc").Find(&records)
	c.JSON(http.StatusOK, records)
}

type statsSummary struct {
	TotalApplied int     `json:"total_applied"`
	TotalWon     int     `json:"total_won"`
	WinRate      float64 `json:"win_rate"`
}

func GetStats(c *gin.Context) {
	userID, _ := c.Get("userID")
	var s statsSummary
	db.DB.Model(&models.Record{}).
		Where("user_id = ?", userID.(uint)).
		Select("COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
		Scan(&s)
	if s.TotalApplied > 0 {
		s.WinRate = float64(s.TotalWon) / float64(s.TotalApplied) * 100
	}
	c.JSON(http.StatusOK, s)
}
