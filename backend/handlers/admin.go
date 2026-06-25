package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// titleKey 同時以 group + single_number 識別一張單曲，避免不同團體的單曲號互相覆蓋
type titleKey struct {
	Group        string
	SingleNumber int
}

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
	Group         string `json:"group"`
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
		Group        string
		SingleNumber int
		SingleName   string
		Count        int64
	}

	// titles 主表已登記的單曲：只要跟登記值不一樣就算問題（含 タイトル未定、空白、打錯字等）。
	// 還沒登記過的單曲（包含所有專輯，因為專輯不會寫入 titles）才退回舊版邏輯：只抓 タイトル未定/空白。
	titles := loadTitleMap()

	countMap := map[titleKey]int64{}
	nameMap := map[titleKey]string{}

	scanIssues := func(model any) {
		var rows []issueRow
		db.DB.Model(model).
			Select(`"group", single_number, single_name, COUNT(*) as count`).
			Group(`"group", single_number, single_name`).
			Scan(&rows)
		for _, r := range rows {
			key := titleKey{Group: r.Group, SingleNumber: r.SingleNumber}
			var isIssue bool
			if registered, ok := titles[key]; ok {
				isIssue = r.SingleName != registered
			} else {
				isIssue = r.SingleName == "" || strings.Contains(r.SingleName, "タイトル未定")
			}
			if !isIssue {
				continue
			}
			countMap[key] += r.Count
			if _, exists := nameMap[key]; !exists {
				nameMap[key] = r.SingleName
			}
		}
	}
	scanIssues(&models.Record{})
	scanIssues(&models.Purchase{})

	// 建議名稱：先查 titles，再從現有正確記錄推測
	type correctRow struct {
		Group        string
		SingleNumber int
		SingleName   string
	}
	var corrects []correctRow
	db.DB.Model(&models.Record{}).
		Select(`"group", single_number, single_name`).
		Where("single_name NOT LIKE ? AND single_name != ''", "%タイトル未定%").
		Group(`"group", single_number, single_name`).
		Scan(&corrects)
	suggestMap := map[titleKey]string{}
	for key, name := range titles {
		suggestMap[key] = name
	}
	for _, r := range corrects {
		key := titleKey{Group: r.Group, SingleNumber: r.SingleNumber}
		if _, exists := suggestMap[key]; !exists {
			suggestMap[key] = r.SingleName
		}
	}

	result := make([]TitleIssue, 0, len(countMap))
	for key, count := range countMap {
		result = append(result, TitleIssue{
			Group:         key.Group,
			SingleNumber:  key.SingleNumber,
			CurrentName:   nameMap[key],
			SuggestedName: suggestMap[key],
			Count:         count,
		})
	}
	// 依團體、單曲號排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].Group != result[j].Group {
			return result[i].Group < result[j].Group
		}
		if result[i].SingleNumber != result[j].SingleNumber {
			return result[i].SingleNumber < result[j].SingleNumber
		}
		return result[i].CurrentName < result[j].CurrentName
	})

	c.JSON(http.StatusOK, result)
}

type KnownTitle struct {
	Group        string `json:"group"`
	SingleNumber int    `json:"single_number"`
	SingleName   string `json:"single_name"`
	Source       string `json:"source"` // correction / records / purchases
}

// albumKey 專輯（single_number == 0）沒有可靠編號可用，只能靠名稱本身互相區分
type albumKey struct {
	Group      string
	SingleName string
}

// GetKnownTitles 列出目前資料庫裡所有已知的單曲名稱（依 group 分組），供管理頁面瀏覽、抓出命名不一致的情況。
// 注意：這只是「DB 裡已經出現過的」單曲，不是官方完整發行紀錄。
// 單曲（single_number > 0）用 (group, single_number) 當鍵；專輯（single_number == 0）沒有可靠編號，改用 (group, single_name) 當鍵，
// 所以同一團體的多張專輯會各自列出，不會互相覆蓋。
func GetKnownTitles(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	type row struct {
		Group        string
		SingleNumber int
		SingleName   string
	}

	nameMap := map[titleKey]string{}
	sourceMap := map[titleKey]string{}
	albumSourceMap := map[albumKey]string{}

	collect := func(rows []row, source string) {
		for _, r := range rows {
			if r.SingleNumber == 0 {
				ak := albumKey{Group: r.Group, SingleName: r.SingleName}
				if _, exists := albumSourceMap[ak]; !exists {
					albumSourceMap[ak] = source
				}
				continue
			}
			key := titleKey{Group: r.Group, SingleNumber: r.SingleNumber}
			nameMap[key] = r.SingleName
			sourceMap[key] = source
		}
	}

	var recRows []row
	db.DB.Model(&models.Record{}).
		Select(`"group", single_number, single_name`).
		Where("single_name NOT LIKE ? AND single_name != ''", "%タイトル未定%").
		Group(`"group", single_number, single_name`).
		Scan(&recRows)
	collect(recRows, "records")

	var purRows []row
	db.DB.Model(&models.Purchase{}).
		Select(`"group", single_number, single_name`).
		Where("single_name NOT LIKE ? AND single_name != ''", "%タイトル未定%").
		Group(`"group", single_number, single_name`).
		Scan(&purRows)
	collect(purRows, "purchases")

	// titles 只會有單曲（專輯沒有可靠編號，不會寫入 titles，見 FixSingleTitle/BulkSetTitles）
	for key, name := range loadTitleMap() {
		nameMap[key] = name
		sourceMap[key] = "correction"
	}

	result := make([]KnownTitle, 0, len(nameMap)+len(albumSourceMap))
	for key, name := range nameMap {
		result = append(result, KnownTitle{
			Group:        key.Group,
			SingleNumber: key.SingleNumber,
			SingleName:   name,
			Source:       sourceMap[key],
		})
	}
	for ak, source := range albumSourceMap {
		result = append(result, KnownTitle{
			Group:        ak.Group,
			SingleNumber: 0,
			SingleName:   ak.SingleName,
			Source:       source,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Group != result[j].Group {
			return result[i].Group < result[j].Group
		}
		if result[i].SingleNumber != result[j].SingleNumber {
			return result[i].SingleNumber < result[j].SingleNumber
		}
		return result[i].SingleName < result[j].SingleName
	})

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

func applyDeleteFilters(q *gorm.DB, c *gin.Context) *gorm.DB {
	if grp := c.Query("group"); grp != "" {
		q = q.Where(`"group" = ?`, grp)
	}
	if sn := c.Query("single_number"); sn != "" {
		q = q.Where("single_number = ?", sn)
	}
	if from := c.Query("date_from"); from != "" {
		q = q.Where("event_date >= ?", from)
	}
	if to := c.Query("date_to"); to != "" {
		q = q.Where("event_date <= ?", to)
	}
	return q
}

func parseAdminTarget(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的使用者 ID"})
		return 0, false
	}
	return uint(id), true
}

func PreviewUserRecords(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	page, pageSize := 1, 50
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	q := applyDeleteFilters(db.DB.Model(&models.Record{}).Where("user_id = ?", targetID), c)
	var total int64
	q.Count(&total)
	var records []models.Record
	q.Order("event_date DESC, member_name ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records)
	c.JSON(http.StatusOK, gin.H{"data": records, "total": total})
}

func PreviewUserFullRecords(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	page, pageSize := 1, 50
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	q := applyDeleteFilters(db.DB.Model(&models.FullRecord{}).Where("user_id = ?", targetID), c)
	var total int64
	q.Count(&total)
	var records []models.FullRecord
	q.Order("event_date DESC, member_name ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&records)
	c.JSON(http.StatusOK, gin.H{"data": records, "total": total})
}

func PreviewUserPurchases(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	page, pageSize := 1, 50
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	q := db.DB.Model(&models.Purchase{}).Where("user_id = ?", targetID)
	if sn := c.Query("single_number"); sn != "" {
		q = q.Where("single_number = ?", sn)
	}
	if from := c.Query("date_from"); from != "" {
		q = q.Where("event_date >= ?", from)
	}
	if to := c.Query("date_to"); to != "" {
		q = q.Where("event_date <= ?", to)
	}
	var total int64
	q.Count(&total)
	var purchases []models.Purchase
	q.Order("event_date DESC, member_name ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&purchases)
	c.JSON(http.StatusOK, gin.H{"data": purchases, "total": total})
}

func DeleteUserRecords(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	q := applyDeleteFilters(db.DB.Where("user_id = ?", targetID), c)
	deleted := q.Delete(&models.Record{}).RowsAffected
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func DeleteUserFullRecords(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	q := applyDeleteFilters(db.DB.Where("user_id = ?", targetID), c)
	deleted := q.Delete(&models.FullRecord{}).RowsAffected
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

func loadTitleMap() map[titleKey]string {
	var corrections []models.Title
	db.DB.Find(&corrections)
	m := make(map[titleKey]string, len(corrections))
	for _, c := range corrections {
		m[titleKey{Group: c.Group, SingleNumber: c.SingleNumber}] = c.SingleName
	}
	return m
}

func FixSingleTitle(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	var req struct {
		Group        string `json:"group" binding:"required"`
		SingleNumber int    `json:"single_number"`
		SingleName   string `json:"single_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 group、single_number 與 single_name"})
		return
	}
	if strings.Contains(req.SingleName, "タイトル未定") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "標題不能包含「タイトル未定」"})
		return
	}

	// 同時更新個握 + 購入（所有使用者），限定同一 group 避免跨團體誤改。
	// 單曲（single_number 唯一對應一張發行物）：直接抓「跟新名稱不一樣」的全部更新，含 タイトル未定/空白/打錯字等任何不一致。
	// 專輯（single_number == 0，同團體可能有多張不同專輯共用這個號碼）：只能抓明顯待修正的 タイトル未定/空白，
	// 不能用「!= 新名稱」，否則會把同團體其他正確命名的專輯也一起改掉。
	recQuery := db.DB.Model(&models.Record{}).Where(`"group" = ? AND single_number = ?`, req.Group, req.SingleNumber)
	purQuery := db.DB.Model(&models.Purchase{}).Where(`"group" = ? AND single_number = ?`, req.Group, req.SingleNumber)
	if req.SingleNumber != 0 {
		recQuery = recQuery.Where("single_name != ?", req.SingleName)
		purQuery = purQuery.Where("single_name != ?", req.SingleName)
	} else {
		recQuery = recQuery.Where("single_name LIKE ? OR single_name = ''", "%タイトル未定%")
		purQuery = purQuery.Where("single_name LIKE ? OR single_name = ''", "%タイトル未定%")
	}
	recResult := recQuery.Update("single_name", req.SingleName)
	purResult := purQuery.Update("single_name", req.SingleName)

	// 專輯（single_number == 0）沒有可靠編號可以當 key，不寫入 titles，
	// 否則之後另一張不同的未定專輯會被誤套用這個名稱
	if req.SingleNumber != 0 {
		db.DB.Where(models.Title{Group: req.Group, SingleNumber: req.SingleNumber}).
			Assign(models.Title{SingleName: req.SingleName}).
			FirstOrCreate(&models.Title{})
	}

	c.JSON(http.StatusOK, gin.H{"updated": recResult.RowsAffected + purResult.RowsAffected})
}

// BulkSetTitles 一次登記多筆已知的單曲名稱（不需要先出現 タイトル未定 問題），
// 同時回填既有的 タイトル未定 紀錄，預防舊單曲之後被任何人抓到時顯示未定。
func BulkSetTitles(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	type titleEntry struct {
		Group        string `json:"group" binding:"required"`
		SingleNumber int    `json:"single_number"`
		SingleName   string `json:"single_name" binding:"required"`
	}
	var req struct {
		Titles []titleEntry `json:"titles" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 titles 陣列"})
		return
	}

	var updated int64
	applied := 0
	for _, t := range req.Titles {
		if t.Group == "" || strings.Contains(t.SingleName, "タイトル未定") || t.SingleName == "" {
			continue
		}

		recQuery := db.DB.Model(&models.Record{}).Where(`"group" = ? AND single_number = ?`, t.Group, t.SingleNumber)
		purQuery := db.DB.Model(&models.Purchase{}).Where(`"group" = ? AND single_number = ?`, t.Group, t.SingleNumber)
		if t.SingleNumber != 0 {
			recQuery = recQuery.Where("single_name != ?", t.SingleName)
			purQuery = purQuery.Where("single_name != ?", t.SingleName)
		} else {
			recQuery = recQuery.Where("single_name LIKE ? OR single_name = ''", "%タイトル未定%")
			purQuery = purQuery.Where("single_name LIKE ? OR single_name = ''", "%タイトル未定%")
		}
		recResult := recQuery.Update("single_name", t.SingleName)
		purResult := purQuery.Update("single_name", t.SingleName)
		updated += recResult.RowsAffected + purResult.RowsAffected

		// 專輯（single_number == 0）沒有可靠編號可以當 key，不寫入 titles（理由同 FixSingleTitle）
		if t.SingleNumber != 0 {
			db.DB.Where(models.Title{Group: t.Group, SingleNumber: t.SingleNumber}).
				Assign(models.Title{SingleName: t.SingleName}).
				FirstOrCreate(&models.Title{})
		}
		applied++
	}

	c.JSON(http.StatusOK, gin.H{"applied": applied, "updated": updated})
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
	db.DB.Where(models.Title{SingleNumber: req.SingleNumber}).
		Assign(models.Title{SingleName: req.SingleName}).
		FirstOrCreate(&models.Title{})

	c.JSON(http.StatusOK, gin.H{"updated": result.RowsAffected})
}
