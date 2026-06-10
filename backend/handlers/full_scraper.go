package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

func PushFullRecords(c *gin.Context) {
	type FullPayload struct {
		OrderID      string  `json:"order_id"`
		SingleNumber int     `json:"single_number"`
		SingleName   string  `json:"single_name"`
		EventType    string  `json:"event_type"`
		Venue        string  `json:"venue"`
		EventDate    string  `json:"event_date"`
		Session      string  `json:"session"`
		MemberName   string  `json:"member_name"`
		AppliedCount int     `json:"applied_count"`
		WonCount     int     `json:"won_count"`
		LotteryRound float64 `json:"lottery_round"`
		SourceURL    string  `json:"source_url"`
	}
	var req struct {
		ScrapeToken string        `json:"scrape_token" binding:"required"`
		Records     []FullPayload `json:"records" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 scrape_token 與 records"})
		return
	}

	var user models.User
	if err := db.DB.Where("scrape_token = ?", req.ScrapeToken).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的 scrape token"})
		return
	}

	newRecords, skipped := 0, 0
	now := time.Now()

	for _, r := range req.Records {
		isSign := strings.Contains(r.OrderID, "_sign")

		if isSign {
			var existing models.SignEvent
			if db.DB.Where("user_id = ? AND order_id = ?", user.ID, r.OrderID).First(&existing).Error == nil {
				skipped++
				continue
			}
			ev := models.SignEvent{
				UserID:       user.ID,
				OrderID:      r.OrderID,
				SingleNumber: r.SingleNumber,
				SingleName:   r.SingleName,
				EventDate:    r.EventDate,
				MemberName:   r.MemberName,
				AppliedCount: r.AppliedCount,
				WonCount:     r.WonCount,
				LotteryRound: r.LotteryRound,
				ScrapedAt:    now,
			}
			if err := db.DB.Create(&ev).Error; err != nil {
				continue
			}
			newRecords++
			continue
		}

		var existing models.FullRecord
		if db.DB.Where("user_id = ? AND order_id = ?", user.ID, r.OrderID).First(&existing).Error == nil {
			skipped++
			continue
		}
		rec := models.FullRecord{
			UserID:       user.ID,
			OrderID:      r.OrderID,
			SingleNumber: r.SingleNumber,
			SingleName:   r.SingleName,
			EventType:    r.EventType,
			Venue:        r.Venue,
			EventDate:    r.EventDate,
			Session:      r.Session,
			MemberName:   r.MemberName,
			AppliedCount: r.AppliedCount,
			WonCount:     r.WonCount,
			LotteryRound: r.LotteryRound,
			SourceURL:    r.SourceURL,
			ScrapedAt:    now,
		}
		if err := db.DB.Create(&rec).Error; err != nil {
			continue
		}
		newRecords++
	}

	c.JSON(http.StatusOK, gin.H{
		"new_records": newRecords,
		"skipped":     skipped,
		"message":     fmt.Sprintf("完成！新增 %d 筆，跳過 %d 筆", newRecords, skipped),
	})
}

func GetFullRecords(c *gin.Context) {
	userID := getUserID(c)

	page, pageSize := 1, 20
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	fmt.Sscan(c.DefaultQuery("page_size", "20"), &pageSize)
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	query := db.DB.Model(&models.FullRecord{}).Where("user_id = ?", userID)
	if m := c.Query("member"); m != "" {
		query = query.Where("member_name = ?", m)
	}
	if et := c.Query("event_type"); et != "" {
		query = query.Where("event_type = ?", et)
	}
	if v := c.Query("venue"); v != "" {
		query = query.Where("venue = ?", v)
	}
	if sn := c.Query("single_number"); sn != "" {
		query = query.Where("single_number = ?", sn)
	}
	if lr := c.Query("lottery_round"); lr != "" {
		query = query.Where("lottery_round = ?", lr)
	}

	var total int64
	query.Count(&total)

	var records []models.FullRecord
	query.Order("event_date DESC, session ASC, member_name ASC").
		Offset(offset).Limit(pageSize).Find(&records)

	c.JSON(http.StatusOK, gin.H{"data": records, "total": total})
}
