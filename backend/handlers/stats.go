package handlers

import (
	"fmt"
	"net/http"

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
	q := db.DB.Model(&models.Record{}).Where("user_id = ?", userID)
	if grp := c.Query("group"); grp != "" {
		q = q.Where(`"group" = ?`, grp)
	}
	var rows []groupedStat
	q.Select("member_name, COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
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
		LotteryRound int     `json:"lottery_round"`
		EventDate    string  `json:"event_date"`
		Session      string  `json:"session"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}

	q := db.DB.Model(&models.Record{}).Where("user_id = ?", userID)
	if grp := c.Query("group"); grp != "" {
		q = q.Where(`"group" = ?`, grp)
	}
	var rows []detailRow
	q.Select("member_name, single_number, MAX(single_name) as single_name, lottery_round, event_date, session, COALESCE(SUM(applied_count),0) as total_applied, COALESCE(SUM(won_count),0) as total_won").
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
if grp := c.Query("group"); grp != "" {
query = query.Where("\"group\" = ?", grp)
}
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

// GetOrderSequenceStats 計算各張應募訂單序號的中選率（依 order_id 排序）
func GetOrderSequenceStats(c *gin.Context) {
	userID := getUserID(c)
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
  WHERE user_id = ?
    AND (? = '' OR member_name = ?)
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
	db.DB.Raw(q, userID, member, member, session, session, round, round).Scan(&rows)

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
