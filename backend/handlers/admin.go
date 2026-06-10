package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

var configuredAdminEmail string

func InitAdmin(email string) {
	configuredAdminEmail = email
}

func checkAdmin(c *gin.Context) bool {
	email, _ := c.Get("email")
	if configuredAdminEmail == "" || email != configuredAdminEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理者限定"})
		return false
	}
	return true
}

type TitleIssue struct {
	SingleNumber  int    `json:"single_number"`
	CurrentName   string `json:"current_name"`
	SuggestedName string `json:"suggested_name"`
	Count         int64  `json:"count"`
}

func GetTitleIssues(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	type issueRow struct {
		SingleNumber int
		SingleName   string
		Count        int64
	}

	// 合併個握 + 購入的 タイトル未定（所有使用者）
	countMap := map[int]int64{}
	nameMap := map[int]string{}

	var recIssues []issueRow
	db.DB.Model(&models.Record{}).
		Select("single_number, single_name, COUNT(*) as count").
		Where("single_name LIKE ?", "%タイトル未定%").
		Group("single_number, single_name").
		Scan(&recIssues)
	for _, r := range recIssues {
		countMap[r.SingleNumber] += r.Count
		nameMap[r.SingleNumber] = r.SingleName
	}

	var purIssues []issueRow
	db.DB.Model(&models.Purchase{}).
		Select("single_number, single_name, COUNT(*) as count").
		Where("single_name LIKE ?", "%タイトル未定%").
		Group("single_number, single_name").
		Scan(&purIssues)
	for _, r := range purIssues {
		countMap[r.SingleNumber] += r.Count
		if _, exists := nameMap[r.SingleNumber]; !exists {
			nameMap[r.SingleNumber] = r.SingleName
		}
	}

	// 建議名稱：先查 title_corrections，再從現有正確記錄推測
	corrections := loadCorrectionMap()
	type correctRow struct {
		SingleNumber int
		SingleName   string
	}
	var corrects []correctRow
	db.DB.Model(&models.Record{}).
		Select("single_number, single_name").
		Where("single_name NOT LIKE ? AND single_name != ''", "%タイトル未定%").
		Group("single_number, single_name").
		Scan(&corrects)
	suggestMap := map[int]string{}
	for sn, name := range corrections {
		suggestMap[sn] = name
	}
	for _, r := range corrects {
		if _, exists := suggestMap[r.SingleNumber]; !exists {
			suggestMap[r.SingleNumber] = r.SingleName
		}
	}

	result := make([]TitleIssue, 0, len(countMap))
	for sn, count := range countMap {
		result = append(result, TitleIssue{
			SingleNumber:  sn,
			CurrentName:   nameMap[sn],
			SuggestedName: suggestMap[sn],
			Count:         count,
		})
	}
	// 依單曲號排序
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].SingleNumber < result[j-1].SingleNumber; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}

	c.JSON(http.StatusOK, result)
}

type AdminUser struct {
	ID          uint       `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	RecordCount int64      `json:"record_count"`
	LastScraped *time.Time `json:"last_scraped"`
}

func GetAdminUsers(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	var users []AdminUser
	db.DB.Model(&models.User{}).
		Select("users.id, users.email, users.name, COUNT(records.id) as record_count, MAX(records.scraped_at) as last_scraped").
		Joins("LEFT JOIN records ON records.user_id = users.id").
		Group("users.id, users.email, users.name").
		Order("users.id").
		Scan(&users)
	c.JSON(http.StatusOK, users)
}

func buildDeleteQuery(c *gin.Context, targetID uint, model interface{}) int64 {
	q := db.DB.Where("user_id = ?", targetID)
	if sn := c.Query("single_number"); sn != "" {
		q = q.Where("single_number = ?", sn)
	}
	if from := c.Query("date_from"); from != "" {
		q = q.Where("event_date >= ?", from)
	}
	if to := c.Query("date_to"); to != "" {
		q = q.Where("event_date <= ?", to)
	}
	return q.Delete(model).RowsAffected
}

func DeleteUserRecords(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的使用者 ID"})
		return
	}
	deleted := buildDeleteQuery(c, uint(targetID), &models.Record{})
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func DeleteUserFullRecords(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的使用者 ID"})
		return
	}
	deleted := buildDeleteQuery(c, uint(targetID), &models.FullRecord{})
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func GetAdminSignEvents(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	page, pageSize := 1, 50
	if p := c.Query("page"); p != "" {
		fmt.Sscan(p, &page)
	}
	if ps := c.Query("page_size"); ps != "" {
		fmt.Sscan(ps, &pageSize)
	}
	if pageSize > 100 {
		pageSize = 100
	}

	type SignEventRow struct {
		ID           uint    `json:"id"`
		UserID       uint    `json:"user_id"`
		UserName     string  `json:"user_name"`
		UserEmail    string  `json:"user_email"`
		SingleNumber int     `json:"single_number"`
		SingleName   string  `json:"single_name"`
		EventDate    string  `json:"event_date"`
		MemberName   string  `json:"member_name"`
		AppliedCount int     `json:"applied_count"`
		WonCount     int     `json:"won_count"`
		LotteryRound float64 `json:"lottery_round"`
	}

	q := db.DB.Table("sign_events").
		Select("sign_events.*, users.name as user_name, users.email as user_email").
		Joins("LEFT JOIN users ON users.id = sign_events.user_id")

	if uid := c.Query("user_id"); uid != "" {
		q = q.Where("sign_events.user_id = ?", uid)
	}
	if m := c.Query("member"); m != "" {
		q = q.Where("sign_events.member_name = ?", m)
	}
	if sn := c.Query("single_number"); sn != "" {
		q = q.Where("sign_events.single_number = ?", sn)
	}

	var total int64
	q.Count(&total)

	var rows []SignEventRow
	q.Order("sign_events.event_date DESC, sign_events.member_name ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&rows)

	c.JSON(http.StatusOK, gin.H{"data": rows, "total": total})
}

func loadCorrectionMap() map[int]string {
	var corrections []models.TitleCorrection
	db.DB.Find(&corrections)
	m := make(map[int]string, len(corrections))
	for _, c := range corrections {
		m[c.SingleNumber] = c.SingleName
	}
	return m
}

func FixSingleTitle(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	var req struct {
		SingleNumber int    `json:"single_number" binding:"required"`
		SingleName   string `json:"single_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 single_number 與 single_name"})
		return
	}
	if strings.Contains(req.SingleName, "タイトル未定") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "標題不能包含「タイトル未定」"})
		return
	}

	// 同時更新個握 + 購入（所有使用者）
	recResult := db.DB.Model(&models.Record{}).
		Where("single_number = ? AND single_name LIKE ?", req.SingleNumber, "%タイトル未定%").
		Update("single_name", req.SingleName)
	purResult := db.DB.Model(&models.Purchase{}).
		Where("single_number = ? AND single_name LIKE ?", req.SingleNumber, "%タイトル未定%").
		Update("single_name", req.SingleName)

	// 儲存對照表供未來自動套用
	db.DB.Where(models.TitleCorrection{SingleNumber: req.SingleNumber}).
		Assign(models.TitleCorrection{SingleName: req.SingleName}).
		FirstOrCreate(&models.TitleCorrection{})

	c.JSON(http.StatusOK, gin.H{"updated": recResult.RowsAffected + purResult.RowsAffected})
}

func GetPurchaseTitleIssues(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	userID := getUserID(c)

	type issueRow struct {
		SingleNumber int
		SingleName   string
		Count        int64
	}
	var issues []issueRow
	db.DB.Model(&models.Purchase{}).
		Select("single_number, single_name, COUNT(*) as count").
		Where("user_id = ? AND single_name LIKE ?", userID, "%タイトル未定%").
		Group("single_number, single_name").
		Order("single_number").
		Scan(&issues)

	type correctRow struct {
		SingleNumber int
		SingleName   string
	}
	var corrects []correctRow
	db.DB.Model(&models.Purchase{}).
		Select("single_number, single_name").
		Where("user_id = ? AND single_name NOT LIKE ? AND single_name != ''", userID, "%タイトル未定%").
		Group("single_number, single_name").
		Scan(&corrects)

	suggestMap := map[int]string{}
	for _, r := range corrects {
		if _, exists := suggestMap[r.SingleNumber]; !exists {
			suggestMap[r.SingleNumber] = r.SingleName
		}
	}

	result := make([]TitleIssue, 0, len(issues))
	for _, iss := range issues {
		result = append(result, TitleIssue{
			SingleNumber:  iss.SingleNumber,
			CurrentName:   iss.SingleName,
			SuggestedName: suggestMap[iss.SingleNumber],
			Count:         iss.Count,
		})
	}
	c.JSON(http.StatusOK, result)
}

func FixPurchaseTitle(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	var req struct {
		SingleNumber int    `json:"single_number" binding:"required"`
		SingleName   string `json:"single_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 single_number 與 single_name"})
		return
	}
	if strings.Contains(req.SingleName, "タイトル未定") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "標題不能包含「タイトル未定」"})
		return
	}

	// 更新所有使用者的購入記錄
	result := db.DB.Model(&models.Purchase{}).
		Where("single_number = ? AND single_name LIKE ?", req.SingleNumber, "%タイトル未定%").
		Update("single_name", req.SingleName)

	// 儲存對照表（與個握共用）
	db.DB.Where(models.TitleCorrection{SingleNumber: req.SingleNumber}).
		Assign(models.TitleCorrection{SingleName: req.SingleName}).
		FirstOrCreate(&models.TitleCorrection{})

	c.JSON(http.StatusOK, gin.H{"updated": result.RowsAffected})
}
