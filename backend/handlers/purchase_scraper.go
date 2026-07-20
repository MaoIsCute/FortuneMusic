package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

type purchaseInput struct {
	EntryID      string `json:"entry_id"`
	Group        string `json:"group"`
	OrderNumber  string `json:"order_number"`
	MemberName   string `json:"member_name"`
	EventDate    string `json:"event_date"`
	Session      string `json:"session"`
	SingleNumber int    `json:"single_number"`
	SingleName   string `json:"single_name"`
	LotteryRound int    `json:"lottery_round"`
	UnitPrice    int    `json:"unit_price"`
	Quantity     int    `json:"quantity"`
	Subtotal     int    `json:"subtotal"`
	AppliedAt    string `json:"applied_at"`
}

func CheckEntries(c *gin.Context) {
	var req struct {
		ScrapeToken string   `json:"scrape_token" binding:"required"`
		EntryIDs    []string `json:"entry_ids" binding:"required"`
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

	var existingIDs []string
	db.DB.Model(&models.Purchase{}).
		Where("user_id = ? AND entry_id IN ?", user.ID, req.EntryIDs).
		Distinct("entry_id").
		Pluck("entry_id", &existingIDs)

	existingSet := map[string]bool{}
	for _, id := range existingIDs {
		existingSet[id] = true
	}

	newIDs := []string{}
	for _, id := range req.EntryIDs {
		if !existingSet[id] {
			newIDs = append(newIDs, id)
		}
	}

	c.JSON(http.StatusOK, gin.H{"new_entry_ids": newIDs})
}

func PushPurchases(c *gin.Context) {
	var req struct {
		ScrapeToken string          `json:"scrape_token" binding:"required"`
		Purchases   []purchaseInput `json:"purchases" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 scrape_token 與 purchases"})
		return
	}

	var user models.User
	if err := db.DB.Where("scrape_token = ?", req.ScrapeToken).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "無效的 scrape token"})
		return
	}

	// event_date 補零成 "YYYY/MM/DD"（見 CLAUDE.md #123），要在算 item_key 之前做，
	// 不然 item_key 裡包的日期段落跟資料庫既有（已回填補零）的 item_key 對不上，
	// 會被誤判成新記錄重複寫入
	for i := range req.Purchases {
		req.Purchases[i].EventDate = normalizeEventDate(req.Purchases[i].EventDate)
	}

	now := time.Now()
	titleMaps := loadTitleMap()

	// 批次查出已存在的 item_key，避免在迴圈中逐筆查詢
	itemKeys := make([]string, 0, len(req.Purchases))
	for _, p := range req.Purchases {
		itemKeys = append(itemKeys, fmt.Sprintf("%s:%s:%s:%s", p.EntryID, p.MemberName, p.EventDate, p.Session))
	}
	existingSet := map[string]bool{}
	if len(itemKeys) > 0 {
		var existing []string
		db.DB.Model(&models.Purchase{}).Where("item_key IN ?", itemKeys).Pluck("item_key", &existing)
		for _, k := range existing {
			existingSet[k] = true
		}
	}

	skipped := 0
	newPurchases := map[string]models.Purchase{} // key 為 item_key，同批重複取最後一筆

	for _, p := range req.Purchases {
		itemKey := fmt.Sprintf("%s:%s:%s:%s", p.EntryID, p.MemberName, p.EventDate, p.Session)
		if existingSet[itemKey] {
			skipped++
			continue
		}

		singleName := p.SingleName
		if p.SingleNumber > 0 {
			// 見 CLAUDE.md #111：只要 titles 表有登記就套用，不再靠名稱格式判斷要不要查表
			if corrected, ok := titleMaps.Singles[titleKey{Group: p.Group, SingleNumber: p.SingleNumber}]; ok {
				singleName = corrected
			}
		} else if p.SingleName != "" {
			// 見 CLAUDE.md #109
			if corrected, ok := titleMaps.Albums[albumCorrKey{Group: p.Group, OrgAlbumName: p.SingleName}]; ok {
				singleName = corrected
			}
		}

		purchase := models.Purchase{
			UserID:       user.ID,
			ItemKey:      itemKey,
			EntryID:      p.EntryID,
			Group:        p.Group,
			OrderNumber:  p.OrderNumber,
			MemberName:   normalizeMember(p.MemberName),
			EventDate:    p.EventDate,
			Session:      p.Session,
			SingleNumber: p.SingleNumber,
			SingleName:   singleName,
			LotteryRound: p.LotteryRound,
			UnitPrice:    p.UnitPrice,
			Quantity:     p.Quantity,
			Subtotal:     p.Subtotal,
			ScrapedAt:    now,
		}
		if p.AppliedAt != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", p.AppliedAt); err == nil {
				purchase.AppliedAt = &t
			}
		}

		newPurchases[itemKey] = purchase
	}

	newCount := 0
	if len(newPurchases) > 0 {
		batch := make([]models.Purchase, 0, len(newPurchases))
		for _, v := range newPurchases {
			batch = append(batch, v)
		}
		if err := db.DB.Create(&batch).Error; err == nil {
			newCount = len(batch)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"new_purchases": newCount,
		"skipped":       skipped,
		"message":       fmt.Sprintf("完成！新增 %d 筆，跳過 %d 筆", newCount, skipped),
	})
}

func GetPurchases(c *gin.Context) {
	userID := getUserID(c)

	page, pageSize := 1, 50
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	fmt.Sscan(c.DefaultQuery("page_size", "50"), &pageSize)
	if pageSize > 100 {
		pageSize = 100
	}

	q := db.DB.Model(&models.Purchase{}).Where("user_id = ?", userID)
	if m := c.Query("member"); m != "" {
		q = q.Where("member_name = ?", m)
	}
	if sn := c.Query("single_number"); sn != "" {
		q = q.Where("single_number = ?", sn)
	}

	var total int64
	q.Count(&total)

	var purchases []models.Purchase
	q.Order("applied_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&purchases)

	c.JSON(http.StatusOK, gin.H{"data": purchases, "total": total})
}

// ─── 統計 ─────────────────────────────────────────────────────────────────────

type PurchaseOverallStats struct {
	TotalAmount   int64 `json:"total_amount"`
	TotalQuantity int64 `json:"total_quantity"`
	PurchaseCount int64 `json:"purchase_count"`
}

func GetPurchaseOverallStats(c *gin.Context) {
	userID := getUserID(c)
	var stats PurchaseOverallStats
	db.DB.Model(&models.Purchase{}).
		Select("COALESCE(SUM(subtotal),0) as total_amount, COALESCE(SUM(quantity),0) as total_quantity, COUNT(*) as purchase_count").
		Where("user_id = ?", userID).
		Scan(&stats)
	c.JSON(http.StatusOK, stats)
}

type PurchaseBySingle struct {
	SingleNumber  int    `json:"single_number"`
	SingleName    string `json:"single_name"`
	LotteryRound  int    `json:"lottery_round"`
	TotalAmount   int64  `json:"total_amount"`
	TotalQuantity int64  `json:"total_quantity"`
}

func GetPurchaseStatsBySingle(c *gin.Context) {
	userID := getUserID(c)
	var rows []PurchaseBySingle
	db.DB.Model(&models.Purchase{}).
		Select("single_number, single_name, lottery_round, SUM(subtotal) as total_amount, SUM(quantity) as total_quantity").
		Where("user_id = ?", userID).
		Group("single_number, single_name, lottery_round").
		Order("single_number DESC, lottery_round ASC").
		Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

type PurchaseByMember struct {
	Group         string `json:"group"`
	MemberName    string `json:"member_name"`
	TotalAmount   int64  `json:"total_amount"`
	TotalQuantity int64  `json:"total_quantity"`
}

func GetPurchaseStatsByMember(c *gin.Context) {
	userID := getUserID(c)
	var rows []PurchaseByMember
	db.DB.Model(&models.Purchase{}).
		Select(`"group", member_name, SUM(subtotal) as total_amount, SUM(quantity) as total_quantity`).
		Where("user_id = ?", userID).
		Group(`"group", member_name`).
		Order("total_amount DESC").
		Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// ─── 樹狀統計（團體 → 單曲 → 抽次 → 成員）──────────────────────────────────────

type treeMember struct {
	MemberName    string `json:"member_name"`
	TotalAmount   int64  `json:"total_amount"`
	TotalQuantity int64  `json:"total_quantity"`
}

type treeRound struct {
	LotteryRound  int          `json:"lottery_round"`
	TotalAmount   int64        `json:"total_amount"`
	TotalQuantity int64        `json:"total_quantity"`
	Members       []treeMember `json:"members"`
}

type treeSingle struct {
	SingleNumber  int         `json:"single_number"`
	SingleName    string      `json:"single_name"`
	TotalAmount   int64       `json:"total_amount"`
	TotalQuantity int64       `json:"total_quantity"`
	Rounds        []treeRound `json:"rounds"`
}

type treeGroup struct {
	Group         string       `json:"group"`
	TotalAmount   int64        `json:"total_amount"`
	TotalQuantity int64        `json:"total_quantity"`
	Singles       []treeSingle `json:"singles"`
}

// 單曲用 (group, single_number) 查發售日；專輯沒有可靠編號，titles 表用抓取時的原始文字
// （org_album_name）當 key，但 purchases.single_name 不一定已經套用過 titles 表的修正——
// 修正是否已經套用取決於這筆資料最後一次寫入/更新的時間點在 #109/#111 那個 bug 修好之前還是之後，
// 修好前寫入的既有資料仍然是原始文字，修好後才會是 titles.single_name 那個修正後的值，兩種都可能出現，
// 所以 org_album_name 跟 single_name 兩個都要能查到同一個 release_date，見 CLAUDE.md #114
type singleReleaseKey struct {
	Group        string
	SingleNumber int
}
type albumReleaseKey struct {
	Group      string
	SingleName string
}

func loadReleaseDateMaps() (map[singleReleaseKey]*time.Time, map[albumReleaseKey]*time.Time) {
	var titleRows []models.Title
	db.DB.Where("release_date IS NOT NULL").Find(&titleRows)
	singles := map[singleReleaseKey]*time.Time{}
	albums := map[albumReleaseKey]*time.Time{}
	for _, t := range titleRows {
		if t.SingleNumber > 0 {
			singles[singleReleaseKey{Group: t.Group, SingleNumber: t.SingleNumber}] = t.ReleaseDate
		} else {
			if t.OrgAlbumName != "" {
				albums[albumReleaseKey{Group: t.Group, SingleName: t.OrgAlbumName}] = t.ReleaseDate
			}
			albums[albumReleaseKey{Group: t.Group, SingleName: t.SingleName}] = t.ReleaseDate
		}
	}
	return singles, albums
}

func GetPurchaseTree(c *gin.Context) {
	userID := getUserID(c)

	var purchases []models.Purchase
	db.DB.Where("user_id = ?", userID).
		Order("lottery_round ASC, member_name ASC").
		Find(&purchases)

	singleReleaseByNum, albumReleaseByName := loadReleaseDateMaps()

	groupOrder := []string{}
	groupMap := map[string]*treeGroup{}
	groupMinTime := map[string]*time.Time{}
	singleOrder := map[string][]string{}
	singleMap := map[string]map[string]*treeSingle{}
	singleMinTime := map[string]map[string]*time.Time{}
	singleReleaseTime := map[string]map[string]*time.Time{}

	for _, p := range purchases {
		g := p.Group
		if _, ok := groupMap[g]; !ok {
			groupMap[g] = &treeGroup{Group: g}
			groupOrder = append(groupOrder, g)
			singleOrder[g] = []string{}
			singleMap[g] = map[string]*treeSingle{}
			singleMinTime[g] = map[string]*time.Time{}
			singleReleaseTime[g] = map[string]*time.Time{}
		}
		grp := groupMap[g]
		grp.TotalAmount += int64(p.Subtotal)
		grp.TotalQuantity += int64(p.Quantity)
		if p.AppliedAt != nil {
			if groupMinTime[g] == nil || p.AppliedAt.Before(*groupMinTime[g]) {
				groupMinTime[g] = p.AppliedAt
			}
		}

		sk := fmt.Sprintf("%d\x00%s", p.SingleNumber, p.SingleName)
		if _, ok := singleMap[g][sk]; !ok {
			singleMap[g][sk] = &treeSingle{
				SingleNumber: p.SingleNumber,
				SingleName:   p.SingleName,
			}
			singleOrder[g] = append(singleOrder[g], sk)
			if p.SingleNumber > 0 {
				singleReleaseTime[g][sk] = singleReleaseByNum[singleReleaseKey{Group: g, SingleNumber: p.SingleNumber}]
			} else {
				singleReleaseTime[g][sk] = albumReleaseByName[albumReleaseKey{Group: g, SingleName: p.SingleName}]
			}
		}
		s := singleMap[g][sk]
		s.TotalAmount += int64(p.Subtotal)
		s.TotalQuantity += int64(p.Quantity)

		if p.AppliedAt != nil {
			if singleMinTime[g][sk] == nil || p.AppliedAt.Before(*singleMinTime[g][sk]) {
				singleMinTime[g][sk] = p.AppliedAt
			}
		}

		// 找或建 round
		ri := -1
		for i := range s.Rounds {
			if s.Rounds[i].LotteryRound == p.LotteryRound {
				ri = i
				break
			}
		}
		if ri == -1 {
			s.Rounds = append(s.Rounds, treeRound{LotteryRound: p.LotteryRound})
			ri = len(s.Rounds) - 1
		}
		s.Rounds[ri].TotalAmount += int64(p.Subtotal)
		s.Rounds[ri].TotalQuantity += int64(p.Quantity)

		// 找或建 member
		mi := -1
		for i := range s.Rounds[ri].Members {
			if s.Rounds[ri].Members[i].MemberName == p.MemberName {
				mi = i
				break
			}
		}
		if mi == -1 {
			s.Rounds[ri].Members = append(s.Rounds[ri].Members, treeMember{MemberName: p.MemberName})
			mi = len(s.Rounds[ri].Members) - 1
		}
		s.Rounds[ri].Members[mi].TotalAmount += int64(p.Subtotal)
		s.Rounds[ri].Members[mi].TotalQuantity += int64(p.Quantity)
	}

	// 依最早購買時間 DESC（新的在前）排序的共用 helper
	byMinTimeDesc := func(keys []string, minTime map[string]*time.Time) {
		sort.Slice(keys, func(i, j int) bool {
			ti := minTime[keys[i]]
			tj := minTime[keys[j]]
			if ti == nil && tj == nil {
				return false
			}
			if ti == nil {
				return false
			}
			if tj == nil {
				return true
			}
			return ti.After(*tj)
		})
	}

	// 單曲層排序：有登記官方發售日的優先照發售日 DESC 排（新的在前），
	// 沒登記過發售日的單曲退回原本「最早購買時間 DESC」的排法，兩種混在一起時
	// 有發售日的一律排在沒有的前面，避免因為缺資料而讓新單曲被舊的購買時間蓋過去
	bySingleOrderDesc := func(keys []string, release map[string]*time.Time, minTime map[string]*time.Time) {
		sort.Slice(keys, func(i, j int) bool {
			ri, rj := release[keys[i]], release[keys[j]]
			if ri != nil || rj != nil {
				if ri == nil {
					return false
				}
				if rj == nil {
					return true
				}
				if !ri.Equal(*rj) {
					return ri.After(*rj)
				}
			}
			ti, tj := minTime[keys[i]], minTime[keys[j]]
			if ti == nil && tj == nil {
				return false
			}
			if ti == nil {
				return false
			}
			if tj == nil {
				return true
			}
			return ti.After(*tj)
		})
	}

	byMinTimeDesc(groupOrder, groupMinTime)

	result := make([]treeGroup, 0, len(groupOrder))
	for _, g := range groupOrder {
		bySingleOrderDesc(singleOrder[g], singleReleaseTime[g], singleMinTime[g])
		grp := groupMap[g]
		for _, sk := range singleOrder[g] {
			grp.Singles = append(grp.Singles, *singleMap[g][sk])
		}
		result = append(result, *grp)
	}
	c.JSON(http.StatusOK, result)
}

// DeleteUserPurchases は admin 用
func DeleteUserPurchases(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	q := applyDeleteFilters(db.DB.Where("user_id = ?", targetID), c)
	deleted := q.Delete(&models.Purchase{}).RowsAffected
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}
