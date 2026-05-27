package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	"fortune-tracker/db"
	"fortune-tracker/models"
	"fortune-tracker/scraper"

	"github.com/gin-gonic/gin"
)

var orderIDRe = regexp.MustCompile(`/apply_detail/([^/#]+)/`)

func extractOrderID(sourceURL string) string {
	m := orderIDRe.FindStringSubmatch(sourceURL)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

type ScrapeRequest struct {
	Cookie     string `json:"cookie"`
	CookieMain string `json:"cookie_main"`
}

// PublicScrape 供瀏覽器擴充功能使用，以 scrape_token 取代 JWT 驗證
func PublicScrape(c *gin.Context) {
	var req struct {
		ScrapeToken   string `json:"scrape_token" binding:"required"`
		Cookie        string `json:"cookie"`         // legacy 單一 cookie
		CookieFortune string `json:"cookie_fortune"` // fortunemusic.jp cookies
		CookieMain    string `json:"cookie_main"`    // main.fortunemusic.jp cookies
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 scrape_token 欄位"})
		return
	}

	var user models.User
	if err := db.DB.Where("scrape_token = ?", req.ScrapeToken).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的 scrape token"})
		return
	}

	// Backward compat: 若新欄位空，用舊的 cookie 當作 fortunemusic cookie
	cookieFortune := req.CookieFortune
	if cookieFortune == "" {
		cookieFortune = req.Cookie
	}

	if len(cookieFortune) < 5 && len(req.CookieMain) < 5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cookie 太短，請確認是否正確"})
		return
	}

	result, err := scraper.Run(user.ID, cookieFortune, req.CookieMain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// PushRecords 接受擴充功能在瀏覽器內爬好的記錄，直接存入 DB。
// 不需要後端自己發 HTTP 請求，完全繞過伺服器 IP 封鎖問題。
func PushRecords(c *gin.Context) {
	type RecordPayload struct {
		MemberName   string `json:"member_name"`
		SingleNumber int    `json:"single_number"`
		SingleName   string `json:"single_name"`
		LotteryRound string `json:"lottery_round"`
		EventDate    string `json:"event_date"`
		Session      string `json:"session"`
		AppliedCount int    `json:"applied_count"`
		WonCount     int    `json:"won_count"`
		SourceURL    string `json:"source_url"`
	}
	var req struct {
		ScrapeToken string          `json:"scrape_token" binding:"required"`
		Records     []RecordPayload `json:"records" binding:"required"`
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
		var existing models.Record
		if db.DB.Where("user_id = ? AND source_url = ?", user.ID, r.SourceURL).First(&existing).Error == nil {
			skipped++
			continue
		}
		rec := models.Record{
			UserID:       user.ID,
			OrderID:      extractOrderID(r.SourceURL),
			SingleNumber: r.SingleNumber,
			SingleName:   r.SingleName,
			LotteryRound: r.LotteryRound,
			MemberName:   r.MemberName,
			EventDate:    r.EventDate,
			Session:      r.Session,
			AppliedCount: r.AppliedCount,
			WonCount:     r.WonCount,
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

// CheckOrders 接受 order ID 列表，回傳哪些是 DB 裡沒有的新訂單。
func CheckOrders(c *gin.Context) {
	var req struct {
		ScrapeToken string   `json:"scrape_token" binding:"required"`
		OrderIDs    []string `json:"order_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 scrape_token 與 order_ids"})
		return
	}
	var user models.User
	if err := db.DB.Where("scrape_token = ?", req.ScrapeToken).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的 scrape token"})
		return
	}

	var foundIDs []string
	db.DB.Model(&models.Record{}).
		Where("user_id = ? AND order_id IN ?", user.ID, req.OrderIDs).
		Distinct("order_id").
		Pluck("order_id", &foundIDs)

	foundSet := make(map[string]bool, len(foundIDs))
	for _, id := range foundIDs {
		foundSet[id] = true
	}

	newIDs, existingIDs := []string{}, []string{}
	for _, id := range req.OrderIDs {
		if foundSet[id] {
			existingIDs = append(existingIDs, id)
		} else {
			newIDs = append(newIDs, id)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"new_order_ids":      newIDs,
		"existing_order_ids": existingIDs,
	})
}

// UpdateTitles 批次更新既有記錄的 single_name / single_number（處理 title 從「未定」變正式名稱）。
func UpdateTitles(c *gin.Context) {
	type TitleUpdate struct {
		OrderID      string `json:"order_id"`
		SingleName   string `json:"single_name"`
		SingleNumber int    `json:"single_number"`
	}
	var req struct {
		ScrapeToken string        `json:"scrape_token" binding:"required"`
		Updates     []TitleUpdate `json:"updates" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 scrape_token 與 updates"})
		return
	}
	var user models.User
	if err := db.DB.Where("scrape_token = ?", req.ScrapeToken).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的 scrape token"})
		return
	}

	updated := 0
	for _, u := range req.Updates {
		if u.SingleName == "" {
			continue
		}
		result := db.DB.Model(&models.Record{}).
			Where("user_id = ? AND order_id = ? AND single_name != ?",
				user.ID, u.OrderID, u.SingleName).
			Updates(map[string]interface{}{
				"single_name":   u.SingleName,
				"single_number": u.SingleNumber,
			})
		updated += int(result.RowsAffected)
	}
	c.JSON(http.StatusOK, gin.H{"updated": updated})
}

func TriggerScrape(c *gin.Context) {
	userID := getUserID(c)

	var req ScrapeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 cookie 欄位"})
		return
	}

	if len(req.Cookie) < 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cookie 太短，請確認是否正確複製"})
		return
	}

	result, err := scraper.Run(userID, req.Cookie, req.CookieMain)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
