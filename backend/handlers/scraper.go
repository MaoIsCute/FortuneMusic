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
		Group        string `json:"group"`
		MemberName   string `json:"member_name"`
		SingleNumber int    `json:"single_number"`
		SingleName   string `json:"single_name"`
		LotteryRound int    `json:"lottery_round"`
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

	now := time.Now()
	titleMaps := loadTitleMap()

	// 批次查出已存在的 source_url，避免在迴圈中逐筆查詢
	sourceURLs := make([]string, 0, len(req.Records))
	for _, r := range req.Records {
		if r.SourceURL != "" {
			sourceURLs = append(sourceURLs, r.SourceURL)
		}
	}
	existingSet := map[string]bool{}
	if len(sourceURLs) > 0 {
		var existing []string
		db.DB.Model(&models.Record{}).Where("source_url IN ?", sourceURLs).Pluck("source_url", &existing)
		for _, u := range existing {
			existingSet[u] = true
		}
	}

	skipped := 0
	toInsertMap := map[string]models.Record{} // key 為 source_url，空字串各自配一個唯一 key 避免互相覆蓋
	noURLSeq := 0
	for _, r := range req.Records {
		if r.SourceURL != "" && existingSet[r.SourceURL] {
			skipped++
			continue
		}
		singleName := r.SingleName
		if r.SingleNumber > 0 {
			// 同一個單曲號，網站在不同頁面/不同時間點可能吐出不只一種「看起來正常」的原始文字
			// （例如刪節號字元跟三個句點的版本都有），不能靠名稱格式判斷對不對，只要 titles 表
			// 有登記就套用，沒登記過就是查表落空、維持原名稱（no-op），見 CLAUDE.md #111
			if corrected, ok := titleMaps.Singles[titleKey{Group: r.Group, SingleNumber: r.SingleNumber}]; ok {
				singleName = corrected
			}
		} else if r.SingleName != "" {
			// 專輯同樣道理，見 CLAUDE.md #109
			if corrected, ok := titleMaps.Albums[albumCorrKey{Group: r.Group, OrgAlbumName: r.SingleName}]; ok {
				singleName = corrected
			}
		}
		rec := models.Record{
			UserID:       user.ID,
			OrderID:      extractOrderID(r.SourceURL),
			Group:        r.Group,
			SingleNumber: r.SingleNumber,
			SingleName:   singleName,
			LotteryRound: r.LotteryRound,
			MemberName:   normalizeMember(r.MemberName),
			EventDate:    r.EventDate,
			Session:      r.Session,
			AppliedCount: r.AppliedCount,
			WonCount:     r.WonCount,
			SourceURL:    r.SourceURL,
			ScrapedAt:    now,
		}
		key := r.SourceURL
		if key == "" {
			noURLSeq++
			key = fmt.Sprintf("__no_url_%d", noURLSeq)
		}
		toInsertMap[key] = rec // 同批重複 source_url 取最後一筆，避免觸發唯一索引衝突
	}

	// 用 transaction 確保同一批全成功或全回滾；批次插入取代逐筆 Create
	newRecords := 0
	if len(toInsertMap) > 0 {
		toInsert := make([]models.Record, 0, len(toInsertMap))
		for _, rec := range toInsertMap {
			toInsert = append(toInsert, rec)
		}
		tx := db.DB.Begin()
		if err := tx.Create(&toInsert).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "寫入失敗，請重試"})
			return
		}
		tx.Commit()
		newRecords = len(toInsert)
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
		Group        string `json:"group"`
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

	titleMaps := loadTitleMap()
	updated := 0
	for _, u := range req.Updates {
		if u.SingleName == "" {
			continue
		}
		singleName := u.SingleName
		if u.SingleNumber > 0 {
			// 見 CLAUDE.md #111：只要 titles 表有登記就套用，不再靠名稱格式判斷要不要查表
			if corrected, ok := titleMaps.Singles[titleKey{Group: u.Group, SingleNumber: u.SingleNumber}]; ok {
				singleName = corrected
			}
		} else if u.SingleName != "" {
			// 見 CLAUDE.md #109
			if corrected, ok := titleMaps.Albums[albumCorrKey{Group: u.Group, OrgAlbumName: u.SingleName}]; ok {
				singleName = corrected
			}
		}
		result := db.DB.Model(&models.Record{}).
			Where("user_id = ? AND order_id = ? AND single_name != ?",
				user.ID, u.OrderID, singleName).
			Updates(map[string]interface{}{
				"single_name":   singleName,
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
