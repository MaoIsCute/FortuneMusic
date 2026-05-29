package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

type purchaseInput struct {
	EntryID      string `json:"entry_id"`
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

	now := time.Now()
	newCount, skipped := 0, 0

	for _, p := range req.Purchases {
		itemKey := fmt.Sprintf("%s:%s:%s:%s", p.EntryID, p.MemberName, p.EventDate, p.Session)

		var exists int64
		db.DB.Model(&models.Purchase{}).
			Where("user_id = ? AND item_key = ?", user.ID, itemKey).
			Count(&exists)
		if exists > 0 {
			skipped++
			continue
		}

		purchase := models.Purchase{
			UserID:       user.ID,
			ItemKey:      itemKey,
			EntryID:      p.EntryID,
			OrderNumber:  p.OrderNumber,
			MemberName:   p.MemberName,
			EventDate:    p.EventDate,
			Session:      p.Session,
			SingleNumber: p.SingleNumber,
			SingleName:   p.SingleName,
			LotteryRound: p.LotteryRound,
			UnitPrice:    p.UnitPrice,
			Quantity:     p.Quantity,
			Subtotal:     p.Subtotal,
			ScrapedAt:    now,
		}
		if p.AppliedAt != "" {
			t, err := time.Parse("2006-01-02 15:04:05", p.AppliedAt)
			if err == nil {
				purchase.AppliedAt = &t
			}
		}

		if err := db.DB.Create(&purchase).Error; err == nil {
			newCount++
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
	MemberName    string `json:"member_name"`
	TotalAmount   int64  `json:"total_amount"`
	TotalQuantity int64  `json:"total_quantity"`
}

func GetPurchaseStatsByMember(c *gin.Context) {
	userID := getUserID(c)
	var rows []PurchaseByMember
	db.DB.Model(&models.Purchase{}).
		Select("member_name, SUM(subtotal) as total_amount, SUM(quantity) as total_quantity").
		Where("user_id = ?", userID).
		Group("member_name").
		Order("total_amount DESC").
		Scan(&rows)
	c.JSON(http.StatusOK, rows)
}

// ─── 樹狀統計（單曲 → 抽次 → 成員）────────────────────────────────────────────

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

func GetPurchaseTree(c *gin.Context) {
	userID := getUserID(c)

	var purchases []models.Purchase
	db.DB.Where("user_id = ?", userID).
		Order("lottery_round ASC, member_name ASC").
		Find(&purchases)

	singleOrder := []string{}
	singleMap := map[string]*treeSingle{}
	singleMinTime := map[string]*time.Time{}

	for _, p := range purchases {
		sk := fmt.Sprintf("%d\x00%s", p.SingleNumber, p.SingleName)
		if _, ok := singleMap[sk]; !ok {
			singleMap[sk] = &treeSingle{
				SingleNumber: p.SingleNumber,
				SingleName:   p.SingleName,
			}
			singleOrder = append(singleOrder, sk)
		}
		s := singleMap[sk]
		s.TotalAmount += int64(p.Subtotal)
		s.TotalQuantity += int64(p.Quantity)

		if p.AppliedAt != nil {
			if singleMinTime[sk] == nil || p.AppliedAt.Before(*singleMinTime[sk]) {
				singleMinTime[sk] = p.AppliedAt
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

	// 依最早購買時間 DESC（新的在前）排序
	sort.Slice(singleOrder, func(i, j int) bool {
		ti := singleMinTime[singleOrder[i]]
		tj := singleMinTime[singleOrder[j]]
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

	result := make([]treeSingle, 0, len(singleOrder))
	for _, k := range singleOrder {
		result = append(result, *singleMap[k])
	}
	c.JSON(http.StatusOK, result)
}

// DeleteUserPurchases は admin 用
func DeleteUserPurchases(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的使用者 ID"})
		return
	}
	deleted := buildDeleteQuery(c, uint(targetID), &models.Purchase{})
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}
