package handlers

import (
	"net/http"
	"strings"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

const adminEmail = "sam6666sunny@gmail.com"

func checkAdmin(c *gin.Context) bool {
	email, _ := c.Get("email")
	if email != adminEmail {
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
	userID := getUserID(c)

	type issueRow struct {
		SingleNumber int
		SingleName   string
		Count        int64
	}
	var issues []issueRow
	db.DB.Model(&models.Record{}).
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
	db.DB.Model(&models.Record{}).
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

func FixSingleTitle(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	userID := getUserID(c)

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

	result := db.DB.Model(&models.Record{}).
		Where("user_id = ? AND single_number = ? AND single_name LIKE ?",
			userID, req.SingleNumber, "%タイトル未定%").
		Update("single_name", req.SingleName)

	c.JSON(http.StatusOK, gin.H{"updated": result.RowsAffected})
}
