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
		Group        string  `json:"group"`
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

	now := time.Now()

	// 先依 order_id 分流，批次查出已存在的記錄，避免在迴圈中逐筆查詢
	var signIDs, fullIDs []string
	for _, r := range req.Records {
		if strings.Contains(r.OrderID, "_sign") {
			signIDs = append(signIDs, r.OrderID)
		} else {
			fullIDs = append(fullIDs, r.OrderID)
		}
	}

	signMap := make(map[string]*models.SignEvent, len(signIDs))
	if len(signIDs) > 0 {
		var existing []models.SignEvent
		db.DB.Where("user_id = ? AND order_id IN ?", user.ID, signIDs).Find(&existing)
		for i := range existing {
			signMap[existing[i].OrderID] = &existing[i]
		}
	}

	fullMap := make(map[string]*models.FullRecord, len(fullIDs))
	if len(fullIDs) > 0 {
		var existing []models.FullRecord
		db.DB.Where("user_id = ? AND order_id IN ?", user.ID, fullIDs).Find(&existing)
		for i := range existing {
			fullMap[existing[i].OrderID] = &existing[i]
		}
	}

	updated, skipped := 0, 0
	newSigns := map[string]models.SignEvent{}
	newFulls := map[string]models.FullRecord{}

	for _, r := range req.Records {
		if strings.Contains(r.OrderID, "_sign") {
			if existing, ok := signMap[r.OrderID]; ok {
				if existing.AppliedCount != r.AppliedCount || existing.WonCount != r.WonCount {
					db.DB.Model(existing).Updates(map[string]any{
						"applied_count": r.AppliedCount,
						"won_count":     r.WonCount,
					})
					updated++
				} else {
					skipped++
				}
				continue
			}
			newSigns[r.OrderID] = models.SignEvent{
				UserID:       user.ID,
				OrderID:      r.OrderID,
				Group:        r.Group,
				SingleNumber: r.SingleNumber,
				SingleName:   r.SingleName,
				EventDate:    r.EventDate,
				MemberName:   normalizeMember(r.MemberName),
				AppliedCount: r.AppliedCount,
				WonCount:     r.WonCount,
				LotteryRound: r.LotteryRound,
				ScrapedAt:    now,
			}
			continue
		}

		if existing, ok := fullMap[r.OrderID]; ok {
			if existing.AppliedCount != r.AppliedCount || existing.WonCount != r.WonCount {
				db.DB.Model(existing).Updates(map[string]any{
					"applied_count": r.AppliedCount,
					"won_count":     r.WonCount,
				})
				updated++
			} else {
				skipped++
			}
			continue
		}
		newFulls[r.OrderID] = models.FullRecord{
			UserID:       user.ID,
			OrderID:      r.OrderID,
			Group:        r.Group,
			SingleNumber: r.SingleNumber,
			SingleName:   r.SingleName,
			EventType:    r.EventType,
			Venue:        r.Venue,
			EventDate:    r.EventDate,
			Session:      r.Session,
			MemberName:   normalizeMember(r.MemberName),
			AppliedCount: r.AppliedCount,
			WonCount:     r.WonCount,
			LotteryRound: r.LotteryRound,
			SourceURL:    r.SourceURL,
			ScrapedAt:    now,
		}
	}

	newRecords := 0
	if len(newSigns) > 0 {
		batch := make([]models.SignEvent, 0, len(newSigns))
		for _, v := range newSigns {
			batch = append(batch, v)
		}
		if err := db.DB.Create(&batch).Error; err == nil {
			newRecords += len(batch)
		}
	}
	if len(newFulls) > 0 {
		batch := make([]models.FullRecord, 0, len(newFulls))
		for _, v := range newFulls {
			batch = append(batch, v)
		}
		if err := db.DB.Create(&batch).Error; err == nil {
			newRecords += len(batch)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"new_records": newRecords,
		"updated":     updated,
		"skipped":     skipped,
		"message":     fmt.Sprintf("完成！新增 %d 筆，更新 %d 筆，跳過 %d 筆", newRecords, updated, skipped),
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
	if grp := c.Query("group"); grp != "" {
		query = query.Where("\"group\" = ?", grp)
	}
	if m := c.Query("member"); m != "" {
		query = query.Where("member_name LIKE ?", "%"+m+"%")
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
