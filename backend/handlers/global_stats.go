package handlers

import (
	"fmt"
	"net/http"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

// 全員統計：跨所有使用者聚合的個握／全握中選率分析。
// 跟個人版統計（stats.go / full_stats.go）的差別只在於拿掉 `user_id = ?` 篩選，
// 顆粒度（含日期/場次/場地）刻意跟個人版一致，不做人數門檻限制；簽名會（SignEvent）不在聚合範圍內。

func recordContributorCount() int64 {
	var n int64
	db.DB.Model(&models.Record{}).Distinct("user_id").Count(&n)
	return n
}

func fullRecordContributorCount() int64 {
	var n int64
	db.DB.Model(&models.FullRecord{}).Distinct("user_id").Count(&n)
	return n
}

// GetGlobalOverallStats 對應個握總表最上方的全體統計卡
func GetGlobalOverallStats(c *gin.Context) {
	var s struct {
		TotalApplied int `json:"total_applied"`
		TotalWon     int `json:"total_won"`
	}
	db.DB.Model(&models.Record{}).
		Select("COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
		Scan(&s)
	winRate := 0.0
	if s.TotalApplied > 0 {
		winRate = float64(s.TotalWon) / float64(s.TotalApplied) * 100
	}
	c.JSON(http.StatusOK, gin.H{
		"total_applied":     s.TotalApplied,
		"total_won":         s.TotalWon,
		"win_rate":          winRate,
		"contributor_count": recordContributorCount(),
	})
}

// GetGlobalDetailStats 對應個握總表的折線圖/長條圖/成員手風琴列表，跟 GetDetailStats 同一套查詢，只是不篩 user_id
func GetGlobalDetailStats(c *gin.Context) {
	type detailRow struct {
		Group        string  `json:"group"`
		MemberName   string  `json:"member_name"`
		SingleNumber int     `json:"single_number"`
		SingleName   string  `json:"single_name"`
		ReleaseDate  string  `json:"release_date"`
		LotteryRound int     `json:"lottery_round"`
		EventDate    string  `json:"event_date"`
		Session      string  `json:"session"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}

	q := db.DB.Model(&models.Record{})
	if grp := c.Query("group"); grp != "" {
		q = q.Where(`records."group" = ?`, grp)
	}
	var rows []detailRow
	q.Joins(`LEFT JOIN titles t ON t."group" = records."group" AND t.single_number = records.single_number AND t.org_album_name = ''`).
		Select(`records."group", records.member_name, records.single_number, MAX(records.single_name) as single_name, ` +
			`COALESCE(TO_CHAR(t.release_date, 'YYYY-MM-DD'), '') as release_date, ` +
			`records.lottery_round, records.event_date, records.session, ` +
			`COALESCE(SUM(records.applied_count),0) as total_applied, COALESCE(SUM(records.won_count),0) as total_won`).
		Group(`records."group", records.member_name, records.single_number, t.release_date, records.lottery_round, records.event_date, records.session`).
		Order(`records."group", records.member_name, records.single_number, records.lottery_round, records.event_date, records.session`).
		Scan(&rows)

	for i := range rows {
		if rows[i].TotalApplied > 0 {
			rows[i].WinRate = float64(rows[i].TotalWon) / float64(rows[i].TotalApplied) * 100
		}
	}
	c.JSON(http.StatusOK, rows)
}

// GetGlobalOrderSequenceStats 跟 GetOrderSequenceStats 同一套邏輯，但不篩 user_id，
// position 因此變成「該場次全體使用者送出順序」而不是單一使用者自己的順序，樣本數更大也更有參考價值
func GetGlobalOrderSequenceStats(c *gin.Context) {
	member := c.Query("member")
	session := c.Query("session")
	round := c.Query("round")

	type result struct {
		Position int     `json:"position"`
		Applied  int     `json:"applied"`
		Won      int     `json:"won"`
		WinRate  float64 `json:"win_rate"`
	}

	const q = `
WITH ranked AS (
  SELECT applied_count, won_count,
    ROW_NUMBER() OVER (
      PARTITION BY event_date, session
      ORDER BY order_id
    ) AS position
  FROM records
  WHERE (? = '' OR member_name = ?)
    AND (? = '' OR session     = ?)
    AND (? = '' OR lottery_round = ?)
)
SELECT
  position,
  SUM(applied_count)                                                        AS applied,
  SUM(won_count)                                                            AS won,
  ROUND(SUM(won_count)::numeric / NULLIF(SUM(applied_count), 0) * 100, 1)  AS win_rate
FROM ranked
GROUP BY position
ORDER BY position`

	var rows []result
	db.DB.Raw(q, member, member, session, session, round, round).Scan(&rows)

	if len(rows) == 0 {
		c.JSON(http.StatusOK, []struct{}{})
		return
	}

	type out struct {
		Position string  `json:"position"`
		Applied  int     `json:"applied"`
		Won      int     `json:"won"`
		WinRate  float64 `json:"win_rate"`
	}
	res := make([]out, len(rows))
	for i, r := range rows {
		res[i] = out{
			Position: fmt.Sprintf("第%d筆", r.Position),
			Applied:  r.Applied,
			Won:      r.Won,
			WinRate:  r.WinRate,
		}
	}
	c.JSON(http.StatusOK, res)
}

// GetGlobalStatsByMember 個握總表新增的「依成員排行榜」，個人版沒有對應端點（單一使用者抽的成員數太少，排行沒有意義）
func GetGlobalStatsByMember(c *gin.Context) {
	q := db.DB.Model(&models.Record{})
	if grp := c.Query("group"); grp != "" {
		q = q.Where(`"group" = ?`, grp)
	}

	type Row struct {
		Group        string  `json:"group"`
		MemberName   string  `json:"member_name"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}
	var rows []Row
	q.Select(`"group", member_name, SUM(applied_count) as total_applied, SUM(won_count) as total_won, ` +
		`ROUND(SUM(won_count)::numeric / NULLIF(SUM(applied_count),0) * 100, 1) as win_rate`).
		Group(`"group", member_name`).
		Order("total_applied DESC").
		Scan(&rows)

	c.JSON(http.StatusOK, rows)
}

// GetGlobalStatsBySingle 個握總表新增的「依單曲排行榜」，同上，個人版沒有對應端點
func GetGlobalStatsBySingle(c *gin.Context) {
	q := db.DB.Model(&models.Record{})
	if grp := c.Query("group"); grp != "" {
		q = q.Where(`records."group" = ?`, grp)
	}

	type Row struct {
		Group        string  `json:"group"`
		SingleNumber int     `json:"single_number"`
		SingleName   string  `json:"single_name"`
		ReleaseDate  string  `json:"release_date"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}
	var rows []Row
	q.Joins(`LEFT JOIN titles t ON t."group" = records."group" AND t.single_number = records.single_number AND t.org_album_name = ''`).
		Select(`records."group", records.single_number, MAX(records.single_name) as single_name, ` +
			`COALESCE(TO_CHAR(t.release_date, 'YYYY-MM-DD'), '') as release_date, ` +
			`SUM(records.applied_count) as total_applied, SUM(records.won_count) as total_won, ` +
			`ROUND(SUM(records.won_count)::numeric / NULLIF(SUM(records.applied_count),0) * 100, 1) as win_rate`).
		Group(`records."group", records.single_number, t.release_date`).
		Order(`records."group", records.single_number DESC`).
		Scan(&rows)

	c.JSON(http.StatusOK, rows)
}

// GetGlobalFullOverallStats 對應全握總表最上方的全體統計卡 + 類型分析
func GetGlobalFullOverallStats(c *gin.Context) {
	var overall struct {
		TotalApplied int `json:"total_applied"`
		TotalWon     int `json:"total_won"`
	}
	db.DB.Model(&models.FullRecord{}).
		Select("COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
		Scan(&overall)

	type TypeVenueRow struct {
		EventType    string  `json:"event_type"`
		Venue        string  `json:"venue"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}
	var byType []TypeVenueRow
	db.DB.Model(&models.FullRecord{}).
		Select("event_type, venue, SUM(applied_count) as total_applied, SUM(won_count) as total_won, " +
			"ROUND(SUM(won_count)::numeric / NULLIF(SUM(applied_count),0) * 100, 1) as win_rate").
		Group("event_type, venue").
		Order("event_type, venue").
		Scan(&byType)

	c.JSON(http.StatusOK, gin.H{
		"overall":           overall,
		"by_type":           byType,
		"contributor_count": fullRecordContributorCount(),
	})
}

// GetGlobalFullStatsByMember 跟 GetFullStatsByMember 同一套查詢，不篩 user_id
func GetGlobalFullStatsByMember(c *gin.Context) {
	query := db.DB.Model(&models.FullRecord{})
	if grp := c.Query("group"); grp != "" {
		query = query.Where(`"group" = ?`, grp)
	}
	if et := c.Query("event_type"); et != "" {
		query = query.Where("event_type = ?", et)
	}
	if v := c.Query("venue"); v != "" {
		query = query.Where("venue = ?", v)
	}
	if region := c.Query("region"); region != "" {
		query = query.Where(venueRegionCaseSQL()+" = ?", region)
	}

	type Row struct {
		Group        string  `json:"group"`
		MemberName   string  `json:"member_name"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}
	var rows []Row
	query.Select(`"group", member_name, SUM(applied_count) as total_applied, SUM(won_count) as total_won, ` +
		`ROUND(SUM(won_count)::numeric / NULLIF(SUM(applied_count),0) * 100, 1) as win_rate`).
		Group(`"group", member_name`).
		Order("total_applied DESC").
		Scan(&rows)

	c.JSON(http.StatusOK, rows)
}

// GetGlobalFullStatsByRegion 跟 GetFullStatsByRegion 同一套查詢，不篩 user_id；
// 集合所有人的資料後，才第一次有機會驗證「地方場中選率是否真的比較高」這種樣本量不足以個人資料判斷的問題
func GetGlobalFullStatsByRegion(c *gin.Context) {
	type Row struct {
		Region       string  `json:"region"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}

	query := db.DB.Model(&models.FullRecord{}).Where("event_type = '実体'")
	if grp := c.Query("group"); grp != "" {
		query = query.Where(`"group" = ?`, grp)
	}

	regionExpr := venueRegionCaseSQL()
	var rows []Row
	query.Select(regionExpr + ` as region, SUM(applied_count) as total_applied, SUM(won_count) as total_won, ` +
		"ROUND(SUM(won_count)::numeric / NULLIF(SUM(applied_count),0) * 100, 1) as win_rate").
		Group(regionExpr).
		Order("region").
		Scan(&rows)

	c.JSON(http.StatusOK, rows)
}

// GetGlobalFullDetailStats 跟 GetFullDetailStats 同一套查詢，不篩 user_id，供全握總表的成員詳細分析使用
func GetGlobalFullDetailStats(c *gin.Context) {
	type Row struct {
		SingleNumber int     `json:"single_number"`
		SingleName   string  `json:"single_name"`
		MemberName   string  `json:"member_name"`
		Venue        string  `json:"venue"`
		Session      string  `json:"session"`
		LotteryRound float64 `json:"lottery_round"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
	}

	query := db.DB.Model(&models.FullRecord{})
	if m := c.Query("member"); m != "" {
		query = query.Where("member_name LIKE ?", "%"+m+"%")
	}
	if et := c.Query("event_type"); et != "" {
		query = query.Where("event_type = ?", et)
	}
	if v := c.Query("venue"); v != "" {
		query = query.Where("venue = ?", v)
	}
	if region := c.Query("region"); region != "" {
		query = query.Where(venueRegionCaseSQL()+" = ?", region)
	}

	var rows []Row
	query.Select("single_number, single_name, member_name, venue, session, lottery_round, SUM(applied_count) as total_applied, SUM(won_count) as total_won").
		Group("single_number, single_name, member_name, venue, session, lottery_round").
		Order("single_number ASC, member_name ASC, venue ASC, session ASC, lottery_round ASC").
		Scan(&rows)

	c.JSON(http.StatusOK, rows)
}
