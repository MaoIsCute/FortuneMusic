package handlers

import (
	"net/http"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

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

	type Row struct {
		MemberName   string  `json:"member_name"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}
	var rows []Row
	query.Select("member_name, SUM(applied_count) as total_applied, SUM(won_count) as total_won, "+
		"ROUND(SUM(won_count)::numeric / NULLIF(SUM(applied_count),0) * 100, 1) as win_rate").
		Group("member_name").
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

	var rows []Row
	query.Select("single_number, single_name, member_name, session, lottery_round, SUM(applied_count) as total_applied, SUM(won_count) as total_won").
		Group("single_number, single_name, member_name, session, lottery_round").
		Order("single_number ASC, member_name ASC, session ASC, lottery_round ASC").
		Scan(&rows)

	c.JSON(http.StatusOK, rows)
}

func GetFullStatsBySingle(c *gin.Context) {
	userID := getUserID(c)

	type Row struct {
		SingleNumber int     `json:"single_number"`
		SingleName   string  `json:"single_name"`
		TotalApplied int     `json:"total_applied"`
		TotalWon     int     `json:"total_won"`
		WinRate      float64 `json:"win_rate"`
	}
	var rows []Row
	q := db.DB.Model(&models.FullRecord{}).Where("user_id = ?", userID)
	if grp := c.Query("group"); grp != "" {
		q = q.Where(`"group" = ?`, grp)
	}
	q.
		Select("single_number, single_name, SUM(applied_count) as total_applied, SUM(won_count) as total_won, "+
			"ROUND(SUM(won_count)::numeric / NULLIF(SUM(applied_count),0) * 100, 1) as win_rate").
		Group("single_number, single_name").
		Order("single_number DESC").
		Scan(&rows)

	c.JSON(http.StatusOK, rows)
}
