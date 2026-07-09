package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

// 場地→地區對照表（関東/地方）。実体場地目前只出現過這幾種，新場地上線後需要在這裡補一筆。
var kantoVenues = []string{"幕張メッセ", "パシフィコ横浜", "東京", "東京ビッグサイト"}
var regionalVenues = []string{"京都パルスプラザ", "ポートメッセなごや", "地方", "インテックス大阪"}

func venueRegionCaseSQL() string {
	quote := func(vs []string) string {
		parts := make([]string, len(vs))
		for i, v := range vs {
			parts[i] = "'" + v + "'"
		}
		return strings.Join(parts, ",")
	}
	return fmt.Sprintf(`CASE WHEN venue IN (%s) THEN '関東' WHEN venue IN (%s) THEN '地方' ELSE 'その他' END`,
		quote(kantoVenues), quote(regionalVenues))
}

func GetFullOverallStats(c *gin.Context) {
	userID := getUserID(c)

	var overall struct {
		TotalApplied int `json:"total_applied"`
		TotalWon     int `json:"total_won"`
	}
	db.DB.Model(&models.FullRecord{}).
		Where("user_id = ?", userID).
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
		Where("user_id = ?", userID).
		Select("event_type, venue, SUM(applied_count) as total_applied, SUM(won_count) as total_won, "+
			"ROUND(SUM(won_count)::numeric / NULLIF(SUM(applied_count),0) * 100, 1) as win_rate").
		Group("event_type, venue").
		Order("event_type, venue").
		Scan(&byType)

	c.JSON(http.StatusOK, gin.H{
		"overall": overall,
		"by_type": byType,
	})
}

func GetFullStatsByMember(c *gin.Context) {
	userID := getUserID(c)

	query := db.DB.Model(&models.FullRecord{}).Where("user_id = ?", userID)
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

func GetFullDetailStats(c *gin.Context) {
	userID := getUserID(c)

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

	query := db.DB.Model(&models.FullRecord{}).Where("user_id = ?", userID)
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

func GetFullStatsByRegion(c *gin.Context) {
	userID := getUserID(c)

	type Row struct {
		Region       string  `json:"region"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}

	query := db.DB.Model(&models.FullRecord{}).Where("user_id = ? AND event_type = '実体'", userID)
	if grp := c.Query("group"); grp != "" {
		query = query.Where(`"group" = ?`, grp)
	}

	regionExpr := venueRegionCaseSQL()
	var rows []Row
	query.Select(regionExpr+` as region, SUM(applied_count) as total_applied, SUM(won_count) as total_won, `+
		"ROUND(SUM(won_count)::numeric / NULLIF(SUM(applied_count),0) * 100, 1) as win_rate").
		Group(regionExpr).
		Order("region").
		Scan(&rows)

	c.JSON(http.StatusOK, rows)
}

func GetFullStatsBySingle(c *gin.Context) {
	userID := getUserID(c)

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
	q := db.DB.Model(&models.FullRecord{}).Where("user_id = ?", userID)
	if grp := c.Query("group"); grp != "" {
		q = q.Where(`full_records."group" = ?`, grp)
	}
	q.Joins(`LEFT JOIN titles t ON t."group" = full_records."group" AND t.single_number = full_records.single_number AND t.org_album_name = ''`).
		Select(`full_records."group", full_records.single_number, full_records.single_name, ` +
			`COALESCE(TO_CHAR(t.release_date, 'YYYY-MM-DD'), '') as release_date, ` +
			`SUM(full_records.applied_count) as total_applied, SUM(full_records.won_count) as total_won, ` +
			`ROUND(SUM(full_records.won_count)::numeric / NULLIF(SUM(full_records.applied_count),0) * 100, 1) as win_rate`).
		Group(`full_records."group", full_records.single_number, full_records.single_name, t.release_date`).
		Order(`full_records."group", full_records.single_number DESC`).
		Scan(&rows)

	c.JSON(http.StatusOK, rows)
}
