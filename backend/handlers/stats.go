package handlers

import (
"fmt"
"net/http"
"regexp"
"sort"

"fortune-tracker/db"
"fortune-tracker/models"

"github.com/gin-gonic/gin"
)

func getUserID(c *gin.Context) uint {
id, _ := c.Get("userID")
return id.(uint)
}

func GetStats(c *gin.Context) {
userID := getUserID(c)

var overall struct {
TotalApplied int
TotalWon     int
TotalRecords int
}
db.DB.Model(&models.Record{}).
Where("user_id = ?", userID).
Select("COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won, COUNT(*) as total_records").
Scan(&overall)

winRate := 0.0
if overall.TotalApplied > 0 {
winRate = float64(overall.TotalWon) / float64(overall.TotalApplied)
}

var memberRows []struct {
MemberName   string
TotalApplied int
TotalWon     int
RecordCount  int
}
db.DB.Model(&models.Record{}).
Where("user_id = ?", userID).
Select("member_name, SUM(applied_count) as total_applied, SUM(won_count) as total_won, COUNT(*) as record_count").
Group("member_name").
Order("total_won DESC").
Scan(&memberRows)

type MemberStat struct {
MemberName   string  `json:"member_name"`
TotalApplied int     `json:"total_applied"`
TotalWon     int     `json:"total_won"`
WinRate      float64 `json:"win_rate"`
RecordCount  int     `json:"record_count"`
}
members := make([]MemberStat, 0, len(memberRows))
for _, r := range memberRows {
wr := 0.0
if r.TotalApplied > 0 {
wr = float64(r.TotalWon) / float64(r.TotalApplied)
}
members = append(members, MemberStat{
MemberName:   r.MemberName,
TotalApplied: r.TotalApplied,
TotalWon:     r.TotalWon,
WinRate:      wr,
RecordCount:  r.RecordCount,
})
}

var dateRows []struct {
EventDate    string
TotalApplied int
TotalWon     int
}
db.DB.Model(&models.Record{}).
Where("user_id = ?", userID).
Select("event_date, SUM(applied_count) as total_applied, SUM(won_count) as total_won").
Group("event_date").
Order("event_date ASC").
Scan(&dateRows)

type DateStat struct {
EventDate    string  `json:"event_date"`
TotalApplied int     `json:"total_applied"`
TotalWon     int     `json:"total_won"`
WinRate      float64 `json:"win_rate"`
}
dates := make([]DateStat, 0, len(dateRows))
for _, r := range dateRows {
wr := 0.0
if r.TotalApplied > 0 {
wr = float64(r.TotalWon) / float64(r.TotalApplied)
}
dates = append(dates, DateStat{
EventDate:    r.EventDate,
TotalApplied: r.TotalApplied,
TotalWon:     r.TotalWon,
WinRate:      wr,
})
}

c.JSON(http.StatusOK, gin.H{
"overall": gin.H{
"total_applied": overall.TotalApplied,
"total_won":     overall.TotalWon,
"total_records": overall.TotalRecords,
"win_rate":      winRate,
},
"by_member": members,
"by_date":   dates,
})
}

// GetOverallStats 對應前端 /api/stats/overall
func GetOverallStats(c *gin.Context) {
userID := getUserID(c)
var s struct {
TotalApplied int `json:"total_applied"`
TotalWon     int `json:"total_won"`
}
db.DB.Model(&models.Record{}).
Where("user_id = ?", userID).
Select("COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
Scan(&s)
winRate := 0.0
if s.TotalApplied > 0 {
winRate = float64(s.TotalWon) / float64(s.TotalApplied) * 100
}
c.JSON(http.StatusOK, gin.H{
"total_applied": s.TotalApplied,
"total_won":     s.TotalWon,
"win_rate":      winRate,
})
}

type groupedStat struct {
MemberName   string  `json:"member_name"`
EventDate    string  `json:"event_date,omitempty"`
Session      string  `json:"session,omitempty"`
TotalApplied int     `json:"total_applied"`
TotalWon     int     `json:"total_won"`
WinRate      float64 `json:"win_rate"`
}

func calcWinRate(rows []groupedStat) []groupedStat {
for i := range rows {
if rows[i].TotalApplied > 0 {
rows[i].WinRate = float64(rows[i].TotalWon) / float64(rows[i].TotalApplied) * 100
}
}
return rows
}

func GetStatsByMember(c *gin.Context) {
userID := getUserID(c)
var rows []groupedStat
db.DB.Model(&models.Record{}).
Where("user_id = ?", userID).
Select("member_name, COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
Group("member_name").
Order("total_won DESC").
Scan(&rows)
c.JSON(http.StatusOK, calcWinRate(rows))
}

func GetStatsByDate(c *gin.Context) {
userID := getUserID(c)
var rows []groupedStat
db.DB.Model(&models.Record{}).
Where("user_id = ?", userID).
Select("member_name, event_date, COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
Group("member_name, event_date").
Order("event_date DESC, member_name").
Scan(&rows)
c.JSON(http.StatusOK, calcWinRate(rows))
}

func GetStatsBySession(c *gin.Context) {
userID := getUserID(c)
var rows []groupedStat
db.DB.Model(&models.Record{}).
Where("user_id = ?", userID).
Select("member_name, session, COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
Group("member_name, session").
Order("member_name, session").
Scan(&rows)
c.JSON(http.StatusOK, calcWinRate(rows))
}

func GetDetailStats(c *gin.Context) {
	userID := getUserID(c)

	type detailRow struct {
		MemberName   string  `json:"member_name"`
		SingleNumber int     `json:"single_number"`
		SingleName   string  `json:"single_name"`
		LotteryRound string  `json:"lottery_round"`
		EventDate    string  `json:"event_date"`
		Session      string  `json:"session"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}

	var rows []detailRow
	db.DB.Model(&models.Record{}).
		Where("user_id = ?", userID).
		Select("member_name, single_number, MAX(single_name) as single_name, lottery_round, event_date, session, COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
		Group("member_name, single_number, lottery_round, event_date, session").
		Order("member_name, single_number, lottery_round, event_date, session").
		Scan(&rows)

	for i := range rows {
		if rows[i].TotalApplied > 0 {
			rows[i].WinRate = float64(rows[i].TotalWon) / float64(rows[i].TotalApplied) * 100
		}
	}
	c.JSON(http.StatusOK, rows)
}

func GetRecords(c *gin.Context) {
userID := getUserID(c)

page, pageSize := 1, 20
if p := c.Query("page"); p != "" {
var n int
if _, err := fmt.Sscanf(p, "%d", &n); err == nil && n > 0 {
page = n
}
}
if ps := c.Query("page_size"); ps != "" {
var n int
if _, err := fmt.Sscanf(ps, "%d", &n); err == nil && n > 0 && n <= 100 {
pageSize = n
}
}

query := db.DB.Model(&models.Record{}).Where("user_id = ?", userID)
if member := c.Query("member"); member != "" {
query = query.Where("member_name = ?", member)
}
if single := c.Query("single"); single != "" {
query = query.Where("single_name = ?", single)
}
if round := c.Query("round"); round != "" {
query = query.Where("lottery_round = ?", round)
}

var total int64
query.Count(&total)

var records []models.Record
query.Order("event_date DESC, session ASC").
Offset((page - 1) * pageSize).Limit(pageSize).
Find(&records)

c.JSON(http.StatusOK, gin.H{
"data":      records,
"total":     total,
"page":      page,
"page_size": pageSize,
})
}

// GetOrderSequenceStats 計算二抽各張訂單序號的中選率
// 從 source_url 提取 order ID，同 (event_date, session) 群組內依序排列
func GetOrderSequenceStats(c *gin.Context) {
	userID := getUserID(c)

	query := db.DB.Model(&models.Record{}).Where("user_id = ?", userID)
	if member := c.Query("member"); member != "" {
		query = query.Where("member_name = ?", member)
	}
	if session := c.Query("session"); session != "" {
		query = query.Where("session = ?", session)
	}
	if round := c.Query("round"); round != "" {
		query = query.Where("lottery_round = ?", round)
	}

	var records []models.Record
	query.Find(&records)

	if len(records) == 0 {
		c.JSON(http.StatusOK, []struct{}{})
		return
	}

	// 依 (event_date, session) 分群
	type groupKey struct{ EventDate, Session string }
	groups := map[groupKey][]models.Record{}
	for _, r := range records {
		key := groupKey{r.EventDate, r.Session}
		groups[key] = append(groups[key], r)
	}

	orderIDRe := regexp.MustCompile(`/apply_detail/([^/]+)/`)

	// 各群組內依 order ID 排序，累加各位置的 applied / won
	posAgg := map[int]struct{ applied, won int }{}
	for _, recs := range groups {
		sort.Slice(recs, func(i, j int) bool {
			mi := orderIDRe.FindStringSubmatch(recs[i].SourceURL)
			mj := orderIDRe.FindStringSubmatch(recs[j].SourceURL)
			idI, idJ := "", ""
			if len(mi) > 1 { idI = mi[1] }
			if len(mj) > 1 { idJ = mj[1] }
			return idI < idJ
		})
		for pos, rec := range recs {
			p := posAgg[pos+1]
			p.applied += rec.AppliedCount
			p.won += rec.WonCount
			posAgg[pos+1] = p
		}
	}

	// 排序位置並組成回傳結果
	positions := make([]int, 0, len(posAgg))
	for p := range posAgg { positions = append(positions, p) }
	sort.Ints(positions)

	type result struct {
		Position string  `json:"position"`
		Applied  int     `json:"applied"`
		Won      int     `json:"won"`
		WinRate  float64 `json:"win_rate"`
	}
	out := make([]result, 0, len(positions))
	for _, p := range positions {
		agg := posAgg[p]
		rate := 0.0
		if agg.applied > 0 {
			rate = float64(agg.won) / float64(agg.applied) * 100
		}
		var wr float64
		fmt.Sscanf(fmt.Sprintf("%.1f", rate), "%f", &wr)
		out = append(out, result{
			Position: fmt.Sprintf("第%d筆", p),
			Applied:  agg.applied,
			Won:      agg.won,
			WinRate:  wr,
		})
	}
	c.JSON(http.StatusOK, out)
}
