package handlers

import (
	"net/http"
	"sort"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

// loadPriceTiersByGroup 讀出所有級距設定，依團體分組並依 FromSingleNumber 由小到大排序，
// 供 resolveUnitPrice 用二分邏輯（由後往前找第一個 <= 目標單曲號的級距）查詢。
func loadPriceTiersByGroup() map[string][]models.PriceTier {
	var tiers []models.PriceTier
	db.DB.Find(&tiers)
	byGroup := map[string][]models.PriceTier{}
	for _, t := range tiers {
		byGroup[t.Group] = append(byGroup[t.Group], t)
	}
	for g := range byGroup {
		sort.Slice(byGroup[g], func(i, j int) bool {
			return byGroup[g][i].FromSingleNumber < byGroup[g][j].FromSingleNumber
		})
	}
	return byGroup
}

// resolveUnitPrice 找「該團體裡 from_single_number <= singleNumber 且最接近」的級距單價。
// 沒有任何級距涵蓋這張單曲（例如比登記過的最早級距還早）時，ok 回傳 false。
func resolveUnitPrice(byGroup map[string][]models.PriceTier, group string, singleNumber int) (price int, ok bool) {
	tiers := byGroup[group]
	for i := len(tiers) - 1; i >= 0; i-- {
		if tiers[i].FromSingleNumber <= singleNumber {
			return tiers[i].UnitPrice, true
		}
	}
	return 0, false
}

// GetPriceTiers 列出目前所有已登記的級距設定，管理者限定。
func GetPriceTiers(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	var tiers []models.PriceTier
	db.DB.Order(`"group", from_single_number`).Find(&tiers)
	c.JSON(http.StatusOK, tiers)
}

// BulkSetPriceTiers 批次登記/更新級距設定（依 group+from_single_number upsert），管理者限定。
func BulkSetPriceTiers(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	type tierEntry struct {
		Group            string `json:"group" binding:"required"`
		FromSingleNumber int    `json:"from_single_number"`
		UnitPrice        int    `json:"unit_price"`
	}
	var req struct {
		Tiers []tierEntry `json:"tiers" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 tiers 陣列"})
		return
	}

	applied := 0
	for _, t := range req.Tiers {
		if t.Group == "" || t.UnitPrice <= 0 {
			continue
		}
		var existing models.PriceTier
		if db.DB.Where(`"group" = ? AND from_single_number = ?`, t.Group, t.FromSingleNumber).First(&existing).Error != nil {
			db.DB.Create(&models.PriceTier{Group: t.Group, FromSingleNumber: t.FromSingleNumber, UnitPrice: t.UnitPrice})
		} else {
			db.DB.Model(&existing).Update("unit_price", t.UnitPrice)
		}
		applied++
	}

	c.JSON(http.StatusOK, gin.H{"applied": applied})
}

// DeletePriceTier 刪除一筆級距設定，管理者限定。
func DeletePriceTier(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	db.DB.Delete(&models.PriceTier{}, c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

type fullSpendingMember struct {
	MemberName    string `json:"member_name"`
	AppliedCount  int    `json:"applied_count"`
	EstimatedCost int    `json:"estimated_cost"`
}
type fullSpendingRound struct {
	LotteryRound  float64              `json:"lottery_round"`
	AppliedCount  int                  `json:"applied_count"`
	EstimatedCost int                  `json:"estimated_cost"`
	Members       []fullSpendingMember `json:"members"`
}
type fullSpendingSingle struct {
	SingleNumber  int                 `json:"single_number"`
	SingleName    string              `json:"single_name"`
	UnitPrice     int                 `json:"unit_price"`
	HasPrice      bool                `json:"has_price"`
	AppliedCount  int                 `json:"applied_count"`
	EstimatedCost int                 `json:"estimated_cost"`
	Rounds        []fullSpendingRound `json:"rounds"`
}
type fullSpendingGroup struct {
	Group         string               `json:"group"`
	AppliedCount  int                  `json:"applied_count"`
	EstimatedCost int                  `json:"estimated_cost"`
	Singles       []fullSpendingSingle `json:"singles"`
}

// GetFullSpendingTree 全握「預估花費」——網站本身抓不到全握的實際購買金額（不像個握花費有
// entry_detail 可以爬到真正的訂單明細），只能用「應募口數 × 登記的單曲單價」回推估算，
// 不是真實花費金額，前端要清楚標示這是預估值。單曲沒有登記過價格時 HasPrice 為 false，
// EstimatedCost 維持 0，不計入上層加總（避免用 0 元誤導使用者以為真的免費）。
// pending（抽選中）的記錄一樣要算，因為應募當下就已經付錢參加抽選，跟有沒有開獎無關。
func GetFullSpendingTree(c *gin.Context) {
	userID := getUserID(c)

	var records []models.FullRecord
	db.DB.Where("user_id = ?", userID).
		Order("single_number DESC, lottery_round ASC, member_name ASC").
		Find(&records)

	priceTiers := loadPriceTiersByGroup()

	groupOrder := []string{}
	groupMap := map[string]*fullSpendingGroup{}
	singleOrder := map[string][]int{}
	singleMap := map[string]map[int]*fullSpendingSingle{}

	for _, r := range records {
		g := r.Group
		if _, ok := groupMap[g]; !ok {
			groupMap[g] = &fullSpendingGroup{Group: g}
			groupOrder = append(groupOrder, g)
			singleOrder[g] = []int{}
			singleMap[g] = map[int]*fullSpendingSingle{}
		}
		grp := groupMap[g]

		if _, ok := singleMap[g][r.SingleNumber]; !ok {
			price, hasPrice := resolveUnitPrice(priceTiers, g, r.SingleNumber)
			singleMap[g][r.SingleNumber] = &fullSpendingSingle{
				SingleNumber: r.SingleNumber,
				SingleName:   r.SingleName,
				UnitPrice:    price,
				HasPrice:     hasPrice,
			}
			singleOrder[g] = append(singleOrder[g], r.SingleNumber)
		}
		single := singleMap[g][r.SingleNumber]

		cost := 0
		if single.HasPrice {
			cost = r.AppliedCount * single.UnitPrice
		}

		single.AppliedCount += r.AppliedCount
		single.EstimatedCost += cost
		grp.AppliedCount += r.AppliedCount
		grp.EstimatedCost += cost

		var round *fullSpendingRound
		for i := range single.Rounds {
			if single.Rounds[i].LotteryRound == r.LotteryRound {
				round = &single.Rounds[i]
				break
			}
		}
		if round == nil {
			single.Rounds = append(single.Rounds, fullSpendingRound{LotteryRound: r.LotteryRound})
			round = &single.Rounds[len(single.Rounds)-1]
		}
		round.AppliedCount += r.AppliedCount
		round.EstimatedCost += cost
		round.Members = append(round.Members, fullSpendingMember{
			MemberName:    r.MemberName,
			AppliedCount:  r.AppliedCount,
			EstimatedCost: cost,
		})
	}

	result := make([]fullSpendingGroup, 0, len(groupOrder))
	for _, g := range groupOrder {
		grp := groupMap[g]
		for _, sn := range singleOrder[g] {
			grp.Singles = append(grp.Singles, *singleMap[g][sn])
		}
		result = append(result, *grp)
	}

	c.JSON(http.StatusOK, result)
}
