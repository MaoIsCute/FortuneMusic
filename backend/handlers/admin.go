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

func normalizeMember(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(name), " ", ""), "　", "")
}

// normalizeEventDate 把擴充功能送來的 "YYYY/M/D"（月、日不補零）轉成補零的 "YYYY/MM/DD"。
// event_date 在 records/full_records/purchases/sign_events/venues 五張表都是純字串欄位，
// 不補零的話字串排序/範圍比較（ORDER BY、>=/<=）跟真正的日期先後順序對不上——例如
// "2026/7/5" 字串上會排在 "2026/7/19" 前面，因為逐字元比較 '5' > '1'；補零成
// "2026/07/05" vs "2026/07/19" 之後，字串順序才會等於日期順序。
// 格式不符合預期（分割後不是三段、或任一段不是數字）就原樣傳回，不強行處理，避免把
// 壞資料吃掉變成看不出來的空值或 0000/00/00。
func normalizeEventDate(s string) string {
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return s
	}
	y, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	d, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return s
	}
	return fmt.Sprintf("%04d/%02d/%02d", y, m, d)
}

// titleKey 同時以 group + single_number 識別一張單曲，避免不同團體的單曲號互相覆蓋
type titleKey struct {
	Group        string
	SingleNumber int
}

// albumCorrKey 以 group + org_album_name 識別一張專輯的改名對照
type albumCorrKey struct {
	Group        string
	OrgAlbumName string
}

// titleMaps 同時持有單曲 map 與專輯改名 map
type titleMaps struct {
	Singles map[titleKey]string    // (group, single_number) → single_name，限 single_number > 0
	Albums  map[albumCorrKey]string // (group, org_album_name) → corrected_name，限 single_number == 0
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
	// 還沒登記過的單曲（包含所有專輯，因為專輯不會寫入 titles）才退回舊版邏輯：只抓 タイトル未定/空白，
	// 但單曲（single_number > 0）額外檢查是否同一單曲號出現兩種以上「看起來正常」的不同名稱互相衝突——
	// 這種情況兩個名稱都不是空白/タイトル未定，舊版邏輯完全偵測不到，資料本身就兜不起來，需要人工選一個正確的。
	// 專輯不適用這個檢查：同團體多張專輯本來就共用 single_number=0，天生就會有「多個不同名稱」，那是正常現象不是衝突。
	maps := loadTitleMap()

	// 依 (group, single_number) 彙總每個不同 single_name 各自的出現次數，橫跨 records + purchases
	nameCounts := map[titleKey]map[string]int64{}

	scanNames := func(model any) {
		var rows []issueRow
		db.DB.Model(model).
			Select(`"group", single_number, single_name, COUNT(*) as count`).
			Group(`"group", single_number, single_name`).
			Scan(&rows)
		for _, r := range rows {
			key := titleKey{Group: r.Group, SingleNumber: r.SingleNumber}
			if nameCounts[key] == nil {
				nameCounts[key] = map[string]int64{}
			}
			nameCounts[key][r.SingleName] += r.Count
		}
	}
	scanNames(&models.Record{})
	scanNames(&models.Purchase{})

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
	for key, name := range maps.Singles {
		suggestMap[key] = name
	}
	for _, r := range corrects {
		key := titleKey{Group: r.Group, SingleNumber: r.SingleNumber}
		if _, exists := suggestMap[key]; !exists {
			suggestMap[key] = r.SingleName
		}
	}

	isRealName := func(name string) bool {
		return name != "" && !strings.Contains(name, "タイトル未定")
	}

	result := make([]TitleIssue, 0, len(nameCounts))
	for key, names := range nameCounts {
		if registered, ok := maps.Singles[key]; ok {
			for name, count := range names {
				if name != registered {
					result = append(result, TitleIssue{
						Group: key.Group, SingleNumber: key.SingleNumber,
						CurrentName: name, SuggestedName: registered, Count: count,
					})
				}
			}
			continue
		}

		if key.SingleNumber > 0 {
			realNameCount := 0
			for name := range names {
				if isRealName(name) {
					realNameCount++
				}
			}
			if realNameCount >= 2 {
				// 同一單曲號出現多種互相衝突的正常名稱，全部列出讓管理者選擇正確版本
				for name, count := range names {
					result = append(result, TitleIssue{
						Group: key.Group, SingleNumber: key.SingleNumber,
						CurrentName: name, SuggestedName: "", Count: count,
					})
				}
				continue
			}
		}

		for name, count := range names {
			if !isRealName(name) {
				result = append(result, TitleIssue{
					Group: key.Group, SingleNumber: key.SingleNumber,
					CurrentName: name, SuggestedName: suggestMap[key], Count: count,
				})
			}
		}
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
	OrgAlbumName string `json:"org_album_name"` // 專輯用：抓取時的原始名稱（作為修正 key）
	Source       string `json:"source"`          // correction / records / purchases
	ReleaseDate  string `json:"release_date"`    // "YYYY-MM-DD" or ""
}

// albumKey 專輯（single_number == 0）沒有可靠編號可用，只能靠名稱本身互相區分
type albumKey struct {
	Group      string
	SingleName string
}

func GetKnownTitles(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	// titles 是主要來源（authoritative）
	var allTitles []models.Title
	db.DB.Find(&allTitles)

	// 記錄哪些單曲/專輯已有 titles 主表資料
	coveredSingles := map[titleKey]bool{}
	type gNamePair struct{ Group, Name string }
	coveredAlbumCorrected := map[gNamePair]bool{} // corrected name（records/purchases 存的名稱）
	coveredAlbumOrg := map[gNamePair]bool{}        // org_album_name（原始抓取名稱）

	result := make([]KnownTitle, 0, len(allTitles)+16)
	for _, t := range allTitles {
		rd := ""
		if t.ReleaseDate != nil {
			rd = t.ReleaseDate.Format("2006-01-02")
		}
		result = append(result, KnownTitle{
			Group:        t.Group,
			SingleNumber: t.SingleNumber,
			SingleName:   t.SingleName,
			OrgAlbumName: t.OrgAlbumName,
			Source:       "correction",
			ReleaseDate:  rd,
		})
		if t.SingleNumber > 0 {
			coveredSingles[titleKey{Group: t.Group, SingleNumber: t.SingleNumber}] = true
		} else {
			coveredAlbumCorrected[gNamePair{t.Group, t.SingleName}] = true
			if t.OrgAlbumName != "" {
				coveredAlbumOrg[gNamePair{t.Group, t.OrgAlbumName}] = true
			}
		}
	}

	// 從 records / purchases 補充未在 titles 的單曲
	type row struct {
		Group        string
		SingleNumber int
		SingleName   string
	}
	singleExtra := map[titleKey]KnownTitle{}

	loadSingles := func(model any, source string) {
		var rows []row
		db.DB.Model(model).
			Select(`"group", single_number, MAX(single_name) as single_name`).
			Where("single_name NOT LIKE ? AND single_name != '' AND single_number > 0", "%タイトル未定%").
			Group(`"group", single_number`).
			Scan(&rows)
		for _, r := range rows {
			key := titleKey{Group: r.Group, SingleNumber: r.SingleNumber}
			if coveredSingles[key] {
				continue
			}
			if _, exists := singleExtra[key]; !exists {
				singleExtra[key] = KnownTitle{Group: r.Group, SingleNumber: r.SingleNumber, SingleName: r.SingleName, Source: source}
			}
		}
	}
	loadSingles(&models.Record{}, "records")
	loadSingles(&models.Purchase{}, "purchases")
	for _, kt := range singleExtra {
		result = append(result, kt)
	}

	// 從 records / purchases 補充未在 titles 的專輯
	albumExtra := map[albumKey]KnownTitle{}

	loadAlbums := func(model any, source string) {
		var rows []row
		db.DB.Model(model).
			Select(`"group", single_name`).
			Where("single_name NOT LIKE ? AND single_name != '' AND single_number = 0", "%タイトル未定%").
			Group(`"group", single_name`).
			Scan(&rows)
		for _, r := range rows {
			ng := gNamePair{r.Group, r.SingleName}
			if coveredAlbumCorrected[ng] || coveredAlbumOrg[ng] {
				continue
			}
			ak := albumKey{Group: r.Group, SingleName: r.SingleName}
			if _, exists := albumExtra[ak]; !exists {
				albumExtra[ak] = KnownTitle{Group: r.Group, SingleNumber: 0, SingleName: r.SingleName, Source: source}
			}
		}
	}
	loadAlbums(&models.Record{}, "records")
	loadAlbums(&models.Purchase{}, "purchases")
	for _, kt := range albumExtra {
		result = append(result, kt)
	}

	groupOrder := map[string]int{"nogizaka46": 0, "sakurazaka46": 1, "hinatazaka46": 2}
	sort.Slice(result, func(i, j int) bool {
		gi, gj := groupOrder[result[i].Group], groupOrder[result[j].Group]
		if gi != gj {
			return gi < gj
		}
		ri, rj := result[i].ReleaseDate, result[j].ReleaseDate
		if ri != rj {
			if ri == "" {
				return false
			}
			if rj == "" {
				return true
			}
			return ri < rj
		}
		if result[i].SingleNumber != result[j].SingleNumber {
			if result[i].SingleNumber == 0 {
				return false
			}
			if result[j].SingleNumber == 0 {
				return true
			}
			return result[i].SingleNumber < result[j].SingleNumber
		}
		return result[i].SingleName < result[j].SingleName
	})
	c.JSON(http.StatusOK, result)
}

type AdminUser struct {
	ID              uint       `json:"id"`
	Email           string     `json:"email"`
	Name            string     `json:"name"`
	RecordCount     int64      `json:"record_count"`
	PurchaseCount   int64      `json:"purchase_count"`
	FullRecordCount int64      `json:"full_record_count"`
	LastScraped     *time.Time `json:"last_scraped"`
}

func GetAdminUsers(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	var users []AdminUser
	// 花費（purchases）/全握（full_records）筆數各自用獨立的相關子查詢算，不能跟 records 一樣直接
	// LEFT JOIN 進同一個 SELECT——一次 JOIN 三張一對多的表會互相撐開成笛卡兒積，COUNT 出來的筆數會不準
	db.DB.Model(&models.User{}).
		Select(`users.id, users.email, users.name,
			(SELECT COUNT(*) FROM records WHERE records.user_id = users.id) as record_count,
			(SELECT COUNT(*) FROM purchases WHERE purchases.user_id = users.id) as purchase_count,
			(SELECT COUNT(*) FROM full_records WHERE full_records.user_id = users.id) as full_record_count,
			(SELECT MAX(records.scraped_at) FROM records WHERE records.user_id = users.id) as last_scraped`).
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
	q := applyDeleteFilters(db.DB.Model(&models.Purchase{}).Where("user_id = ?", targetID), c)
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

func PreviewUserSignEvents(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	page, pageSize := 1, 50
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	q := applyDeleteFilters(db.DB.Model(&models.SignEvent{}).Where("user_id = ?", targetID), c)
	var total int64
	q.Count(&total)
	var rows []models.SignEvent
	q.Order("event_date DESC, member_name ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows)
	c.JSON(http.StatusOK, gin.H{"data": rows, "total": total})
}

func DeleteUserSignEvents(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	q := applyDeleteFilters(db.DB.Where("user_id = ?", targetID), c)
	deleted := q.Delete(&models.SignEvent{}).RowsAffected
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

// applyPrizeDeleteFilters 跟 applyDeleteFilters 分開一份，因為 prizes 表沒有 event_date 欄位，
// 不能套用 date_from/date_to（那兩個條件會對不存在的欄位下 SQL 直接出錯）
func applyPrizeDeleteFilters(q *gorm.DB, c *gin.Context) *gorm.DB {
	if grp := c.Query("group"); grp != "" {
		q = q.Where(`"group" = ?`, grp)
	}
	if sn := c.Query("single_number"); sn != "" {
		q = q.Where("single_number = ?", sn)
	}
	return q
}

func PreviewUserPrizes(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	page, pageSize := 1, 50
	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
	q := applyPrizeDeleteFilters(db.DB.Model(&models.Prize{}).Where("user_id = ?", targetID), c)
	var total int64
	q.Count(&total)
	var rows []models.Prize
	q.Order("single_number DESC, prize_code ASC, member_name ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows)
	c.JSON(http.StatusOK, gin.H{"data": rows, "total": total})
}

func DeleteUserPrizes(c *gin.Context) {
	if !checkAdmin(c) { return }
	targetID, ok := parseAdminTarget(c)
	if !ok { return }
	q := applyPrizeDeleteFilters(db.DB.Where("user_id = ?", targetID), c)
	deleted := q.Delete(&models.Prize{}).RowsAffected
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
		Group        string  `json:"group"`
		SingleNumber int     `json:"single_number"`
		SingleName   string  `json:"single_name"`
		Venue        string  `json:"venue"`
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
	if grp := c.Query("group"); grp != "" {
		q = q.Where("sign_events.\"group\" = ?", grp)
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

// GetAdminPrizes 列出所有使用者的「商品抽選」申請紀錄，比照 GetAdminSignEvents 同一套模式
func GetAdminPrizes(c *gin.Context) {
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

	type PrizeRow struct {
		ID           uint   `json:"id"`
		UserID       uint   `json:"user_id"`
		UserName     string `json:"user_name"`
		UserEmail    string `json:"user_email"`
		Group        string `json:"group"`
		SingleNumber int    `json:"single_number"`
		PrizeCode    string `json:"prize_code"`
		MemberName   string `json:"member_name"`
		AppliedCount int    `json:"applied_count"`
		WonStatus    string `json:"won_status"`
	}

	q := db.DB.Table("prizes").
		Select("prizes.*, users.name as user_name, users.email as user_email").
		Joins("LEFT JOIN users ON users.id = prizes.user_id")

	if uid := c.Query("user_id"); uid != "" {
		q = q.Where("prizes.user_id = ?", uid)
	}
	if grp := c.Query("group"); grp != "" {
		q = q.Where("prizes.\"group\" = ?", grp)
	}
	if m := c.Query("member"); m != "" {
		q = q.Where("prizes.member_name = ?", m)
	}
	if pc := c.Query("prize_code"); pc != "" {
		q = q.Where("prizes.prize_code = ?", pc)
	}

	var total int64
	q.Count(&total)

	var rows []PrizeRow
	q.Order("prizes.single_number DESC, prizes.prize_code ASC, prizes.member_name ASC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&rows)

	c.JSON(http.StatusOK, gin.H{"data": rows, "total": total})
}

func loadTitleMap() titleMaps {
	var titles []models.Title
	db.DB.Find(&titles)
	m := titleMaps{
		Singles: make(map[titleKey]string),
		Albums:  make(map[albumCorrKey]string),
	}
	for _, t := range titles {
		if t.SingleNumber > 0 {
			m.Singles[titleKey{Group: t.Group, SingleNumber: t.SingleNumber}] = t.SingleName
		} else if t.OrgAlbumName != "" {
			m.Albums[albumCorrKey{Group: t.Group, OrgAlbumName: t.OrgAlbumName}] = t.SingleName
		}
	}
	return m
}

// looksLikeDisplayFormat 偵測是不是誤把「畫面顯示格式」當成原始名稱存進來。
// RecordsView.vue 等前端頁面的 formatSingle() 只在顯示時把「Nthアルバム」轉成中文「N專」、
// 「アルバム」轉成「專輯」，titles/records/purchases 的 single_name 欄位定位是原始日文名稱，
// 中文轉換不該存進資料庫——否則畫面顯示層的規則以後要調整時，已經被中文化的舊資料不會再被
// formatSingle() 處理到，變成撿不乾淨的技術債（見 CLAUDE.md #110）。
// 「專」是繁體中文字（U+5C08），跟日文的「専」（U+5C02，如「専用」）是不同字，用它當偵測特徵不會誤判到合法的日文名稱。
func looksLikeDisplayFormat(name string) bool {
	return strings.Contains(name, "專")
}

func FixSingleTitle(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	var req struct {
		Group        string `json:"group" binding:"required"`
		SingleNumber int    `json:"single_number"`
		SingleName   string `json:"single_name" binding:"required"`
		OrgAlbumName string `json:"org_album_name"` // 專輯用：原始抓取名稱作為修正 key
		ReleaseDate  string `json:"release_date"`   // "YYYY-MM-DD" or ""
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 group、single_number 與 single_name"})
		return
	}
	if strings.Contains(req.SingleName, "タイトル未定") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "標題不能包含「タイトル未定」"})
		return
	}
	if looksLikeDisplayFormat(req.SingleName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "標題看起來是畫面顯示格式（例如「5專」/「專輯」），請填入原始日文格式（例如「5thアルバム『標題』」）；中文是前端顯示時才轉換的，不應該存進資料庫"})
		return
	}

	var rd *time.Time
	if req.ReleaseDate != "" {
		if t, err := time.Parse("2006-01-02", req.ReleaseDate); err == nil {
			rd = &t
		}
	}

	// upsertTitle 以 (group, single_number, org_album_name) 為 key，若存在則更新，否則建立
	upsertTitle := func(group string, singleNumber int, orgAlbumName, singleName string, releaseDate *time.Time) {
		var t models.Title
		if db.DB.Where(`"group" = ? AND single_number = ? AND org_album_name = ?`, group, singleNumber, orgAlbumName).
			First(&t).Error != nil {
			db.DB.Create(&models.Title{
				Group: group, SingleNumber: singleNumber, OrgAlbumName: orgAlbumName,
				SingleName: singleName, ReleaseDate: releaseDate,
			})
		} else {
			db.DB.Model(&t).Updates(map[string]interface{}{"single_name": singleName, "release_date": releaseDate})
		}
	}

	var recAffected, purAffected int64

	if req.SingleNumber != 0 {
		recAffected = db.DB.Model(&models.Record{}).
			Where(`"group" = ? AND single_number = ? AND single_name != ?`, req.Group, req.SingleNumber, req.SingleName).
			Update("single_name", req.SingleName).RowsAffected
		purAffected = db.DB.Model(&models.Purchase{}).
			Where(`"group" = ? AND single_number = ? AND single_name != ?`, req.Group, req.SingleNumber, req.SingleName).
			Update("single_name", req.SingleName).RowsAffected
		upsertTitle(req.Group, req.SingleNumber, "", req.SingleName, rd)
	} else {
		orgName := req.OrgAlbumName
		oldName := orgName
		var existing models.Title
		if db.DB.Where(`"group" = ? AND single_number = 0 AND org_album_name = ?`, req.Group, orgName).First(&existing).Error == nil {
			oldName = existing.SingleName
		}
		recAffected = db.DB.Model(&models.Record{}).
			Where(`"group" = ? AND single_number = 0 AND single_name = ? AND single_name != ?`, req.Group, oldName, req.SingleName).
			Update("single_name", req.SingleName).RowsAffected
		purAffected = db.DB.Model(&models.Purchase{}).
			Where(`"group" = ? AND single_number = 0 AND single_name = ? AND single_name != ?`, req.Group, oldName, req.SingleName).
			Update("single_name", req.SingleName).RowsAffected
		if orgName != "" && !strings.Contains(orgName, "タイトル未定") {
			upsertTitle(req.Group, 0, orgName, req.SingleName, rd)
		}
	}

	c.JSON(http.StatusOK, gin.H{"updated": recAffected + purAffected})
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
		if t.Group == "" || strings.Contains(t.SingleName, "タイトル未定") || t.SingleName == "" || looksLikeDisplayFormat(t.SingleName) {
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

		// 批次登記單曲寫入 titles；專輯沒有 org_album_name 資訊，不存入 titles（只回填既有記錄）
		if t.SingleNumber != 0 {
			db.DB.Where(models.Title{Group: t.Group, SingleNumber: t.SingleNumber, OrgAlbumName: ""}).
				Assign(models.Title{SingleName: t.SingleName}).
				FirstOrCreate(&models.Title{})
		}
		applied++
	}

	c.JSON(http.StatusOK, gin.H{"applied": applied, "updated": updated})
}

// venueKey 以 group + single_number + event_date 識別一個実体場次，
// 因為同一張單可能跨多個日期在不同場地舉辦，不能只用 (group, single_number) 當 key
type venueKey struct {
	Group        string
	SingleNumber int
	EventDate    string
}

func loadVenueMap() map[venueKey]string {
	var venues []models.Venue
	db.DB.Find(&venues)
	m := make(map[venueKey]string, len(venues))
	for _, v := range venues {
		m[venueKey{Group: v.Group, SingleNumber: v.SingleNumber, EventDate: v.EventDate}] = v.VenueName
	}
	return m
}

type VenueIssue struct {
	Group          string `json:"group"`
	SingleNumber   int    `json:"single_number"`
	SingleName     string `json:"single_name"`
	EventDate      string `json:"event_date"`
	CurrentVenue   string `json:"current_venue"`
	Count          int64  `json:"count"`
	SuggestedVenue string `json:"suggested_venue"`
}

// GetVenueIssues 列出実体場次裡場地有問題的 (group, single_number, event_date) 組合：
// 1) 場地空白 — 早期抓取版本沒有解析場地欄位留下的舊資料缺口，來源網站也無法回溯，只能人工登記
// 2) 場地跟 venues 主表已登記的值不一致 — PushFullRecords 只在既有場地為空白時才會套用登記值（見 full_scraper.go），
//    非空白但跟登記值不同的場地文字（打錯字、來源網站文字變動）永遠不會被自動覆蓋，需要人工比對
// 3) 還沒登記過的場次，同時出現 2 種以上互相衝突的非空白場地文字 — 無法判斷哪個正確，全部列出讓管理者挑選
//    （跟 GetTitleIssues 對單曲名稱衝突的偵測邏輯同一套）
func GetVenueIssues(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	venueMap := loadVenueMap()

	type blankRow struct {
		Group        string
		SingleNumber int
		SingleName   string
		EventDate    string
		Count        int64
	}
	var blanks []blankRow
	db.DB.Model(&models.FullRecord{}).
		Select(`"group", single_number, single_name, event_date, COUNT(*) as count`).
		Where(`event_type = '実体' AND (venue IS NULL OR venue = '')`).
		Group(`"group", single_number, single_name, event_date`).
		Scan(&blanks)

	result := make([]VenueIssue, 0, len(blanks))
	blankIndex := map[venueKey]int{} // key → result 裡的 index，讓簽名會的空白筆數可以併進同一列，不要重複列出同一個場次
	for _, r := range blanks {
		key := venueKey{Group: r.Group, SingleNumber: r.SingleNumber, EventDate: r.EventDate}
		blankIndex[key] = len(result)
		result = append(result, VenueIssue{
			Group: r.Group, SingleNumber: r.SingleNumber, SingleName: r.SingleName, EventDate: r.EventDate,
			Count: r.Count, SuggestedVenue: venueMap[key],
		})
	}

	// 簽名會（SignEvent）跟全握一樣有 event_type（見 #121），線上場次本來就沒有場地，
	// 只有実体場次的場地空白才算問題；場地空白一樣要列進問題列表，跟全握共用同一套 venueMap
	// 反推建議值；同一個 (group, single_number, event_date) 如果全握那邊已經列過，直接把筆數
	// 併進去，不要讓管理者看到同一個場次出現兩列
	var signBlanks []blankRow
	db.DB.Model(&models.SignEvent{}).
		Select(`"group", single_number, single_name, event_date, COUNT(*) as count`).
		Where(`event_type = '実体' AND (venue IS NULL OR venue = '')`).
		Group(`"group", single_number, single_name, event_date`).
		Scan(&signBlanks)
	for _, r := range signBlanks {
		key := venueKey{Group: r.Group, SingleNumber: r.SingleNumber, EventDate: r.EventDate}
		if idx, ok := blankIndex[key]; ok {
			result[idx].Count += r.Count
			continue
		}
		blankIndex[key] = len(result)
		result = append(result, VenueIssue{
			Group: r.Group, SingleNumber: r.SingleNumber, SingleName: r.SingleName, EventDate: r.EventDate,
			Count: r.Count, SuggestedVenue: venueMap[key],
		})
	}

	type venueRow struct {
		Group        string
		SingleNumber int
		SingleName   string
		EventDate    string
		Venue        string
		Count        int64
	}
	var venueRows []venueRow
	db.DB.Model(&models.FullRecord{}).
		Select(`"group", single_number, single_name, event_date, venue, COUNT(*) as count`).
		Where(`event_type = '実体' AND venue IS NOT NULL AND venue != ''`).
		Group(`"group", single_number, single_name, event_date, venue`).
		Scan(&venueRows)

	distinctVenues := map[venueKey]map[string]bool{}
	for _, r := range venueRows {
		key := venueKey{Group: r.Group, SingleNumber: r.SingleNumber, EventDate: r.EventDate}
		if distinctVenues[key] == nil {
			distinctVenues[key] = map[string]bool{}
		}
		distinctVenues[key][r.Venue] = true
	}

	for _, r := range venueRows {
		key := venueKey{Group: r.Group, SingleNumber: r.SingleNumber, EventDate: r.EventDate}
		if registered, ok := venueMap[key]; ok {
			if r.Venue != registered {
				result = append(result, VenueIssue{
					Group: r.Group, SingleNumber: r.SingleNumber, SingleName: r.SingleName, EventDate: r.EventDate,
					CurrentVenue: r.Venue, Count: r.Count, SuggestedVenue: registered,
				})
			}
			continue
		}
		if len(distinctVenues[key]) >= 2 {
			result = append(result, VenueIssue{
				Group: r.Group, SingleNumber: r.SingleNumber, SingleName: r.SingleName, EventDate: r.EventDate,
				CurrentVenue: r.Venue, Count: r.Count, SuggestedVenue: "",
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Group != result[j].Group {
			return result[i].Group < result[j].Group
		}
		if result[i].SingleNumber != result[j].SingleNumber {
			return result[i].SingleNumber < result[j].SingleNumber
		}
		if result[i].EventDate != result[j].EventDate {
			return result[i].EventDate < result[j].EventDate
		}
		return result[i].CurrentVenue < result[j].CurrentVenue
	})

	c.JSON(http.StatusOK, result)
}

// FixVenue 登記單一 (group, single_number, event_date) 的場地，同時回填既有跟登記值不同的紀錄（含空白與打錯字），
// 並寫入 venues 表供未來新匯入的同場次資料自動套用
func FixVenue(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	var req struct {
		Group        string `json:"group" binding:"required"`
		SingleNumber int    `json:"single_number"`
		EventDate    string `json:"event_date" binding:"required"`
		Venue        string `json:"venue" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 group、event_date 與 venue"})
		return
	}

	var v models.Venue
	if db.DB.Where(`"group" = ? AND single_number = ? AND event_date = ?`, req.Group, req.SingleNumber, req.EventDate).
		First(&v).Error != nil {
		db.DB.Create(&models.Venue{
			Group: req.Group, SingleNumber: req.SingleNumber, EventDate: req.EventDate, VenueName: req.Venue,
		})
	} else {
		db.DB.Model(&v).Update("venue_name", req.Venue)
	}

	updated := db.DB.Model(&models.FullRecord{}).
		Where(`"group" = ? AND single_number = ? AND event_date = ? AND event_type = '実体' AND (venue IS NULL OR venue != ?)`,
			req.Group, req.SingleNumber, req.EventDate, req.Venue).
		Update("venue", req.Venue).RowsAffected
	// 簽名會場次跟全握共用同一個 (group, single_number, event_date)，一起回填（見 #119）；
	// 線上場次不該被塞進場地，一樣加 event_type = '実体' 限制（見 #121）
	updated += db.DB.Model(&models.SignEvent{}).
		Where(`"group" = ? AND single_number = ? AND event_date = ? AND event_type = '実体' AND (venue IS NULL OR venue != ?)`,
			req.Group, req.SingleNumber, req.EventDate, req.Venue).
		Update("venue", req.Venue).RowsAffected

	c.JSON(http.StatusOK, gin.H{"updated": updated})
}

// BulkSetVenues 一次登記多筆已知的場地（不需要先出現問題列表），
// 同時回填既有跟登記值不同的紀錄（含空白與打錯字），並寫入 venues 表供未來自動套用
func BulkSetVenues(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	type venueEntry struct {
		Group        string `json:"group" binding:"required"`
		SingleNumber int    `json:"single_number"`
		EventDate    string `json:"event_date" binding:"required"`
		Venue        string `json:"venue" binding:"required"`
	}
	var req struct {
		Venues []venueEntry `json:"venues" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 venues 陣列"})
		return
	}

	var updated int64
	applied := 0
	for _, v := range req.Venues {
		if v.Group == "" || v.EventDate == "" || v.Venue == "" {
			continue
		}

		db.DB.Where(models.Venue{Group: v.Group, SingleNumber: v.SingleNumber, EventDate: v.EventDate}).
			Assign(models.Venue{VenueName: v.Venue}).
			FirstOrCreate(&models.Venue{})

		result := db.DB.Model(&models.FullRecord{}).
			Where(`"group" = ? AND single_number = ? AND event_date = ? AND event_type = '実体' AND (venue IS NULL OR venue != ?)`,
				v.Group, v.SingleNumber, v.EventDate, v.Venue).
			Update("venue", v.Venue)
		updated += result.RowsAffected
		// 簽名會場次跟全握共用同一個 (group, single_number, event_date)，一起回填（見 #119）；
		// 線上場次不該被塞進場地，一樣加 event_type = '実体' 限制（見 #121）
		signResult := db.DB.Model(&models.SignEvent{}).
			Where(`"group" = ? AND single_number = ? AND event_date = ? AND event_type = '実体' AND (venue IS NULL OR venue != ?)`,
				v.Group, v.SingleNumber, v.EventDate, v.Venue).
			Update("venue", v.Venue)
		updated += signResult.RowsAffected
		applied++
	}

	c.JSON(http.StatusOK, gin.H{"applied": applied, "updated": updated})
}

type KnownVenue struct {
	Group        string `json:"group"`
	SingleNumber int    `json:"single_number"`
	SingleName   string `json:"single_name"`
	EventDate    string `json:"event_date"`
	Venue        string `json:"venue"`
	Source       string `json:"source"` // correction / records
}

// GetKnownVenues 列出目前已知的場地對照：venues 表登記過的，加上 full_records 裡已經有場地文字的實際資料
func GetKnownVenues(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}

	var registered []models.Venue
	db.DB.Find(&registered)

	// venues 表本身沒有存單曲名稱，補查 full_records 裡同 (group, single_number) 的名稱供顯示
	type singleNameRow struct {
		Group        string
		SingleNumber int
		SingleName   string
	}
	var singleNameRows []singleNameRow
	db.DB.Model(&models.FullRecord{}).
		Select(`"group", single_number, MAX(single_name) as single_name`).
		Where("single_name != ''").
		Group(`"group", single_number`).
		Scan(&singleNameRows)
	singleNameMap := map[titleKey]string{}
	for _, r := range singleNameRows {
		singleNameMap[titleKey{Group: r.Group, SingleNumber: r.SingleNumber}] = r.SingleName
	}

	covered := map[venueKey]bool{}
	result := make([]KnownVenue, 0, len(registered)+16)
	for _, v := range registered {
		result = append(result, KnownVenue{
			Group: v.Group, SingleNumber: v.SingleNumber, EventDate: v.EventDate, Venue: v.VenueName, Source: "correction",
			SingleName: singleNameMap[titleKey{Group: v.Group, SingleNumber: v.SingleNumber}],
		})
		covered[venueKey{Group: v.Group, SingleNumber: v.SingleNumber, EventDate: v.EventDate}] = true
	}

	type row struct {
		Group        string
		SingleNumber int
		SingleName   string
		EventDate    string
		Venue        string
	}
	var rows []row
	db.DB.Model(&models.FullRecord{}).
		Select(`"group", single_number, MAX(single_name) as single_name, event_date, venue`).
		Where(`event_type = '実体' AND venue IS NOT NULL AND venue != ''`).
		Group(`"group", single_number, event_date, venue`).
		Scan(&rows)
	for _, r := range rows {
		key := venueKey{Group: r.Group, SingleNumber: r.SingleNumber, EventDate: r.EventDate}
		if covered[key] {
			continue
		}
		covered[key] = true
		result = append(result, KnownVenue{
			Group: r.Group, SingleNumber: r.SingleNumber, SingleName: r.SingleName,
			EventDate: r.EventDate, Venue: r.Venue, Source: "records",
		})
	}

	groupOrder := map[string]int{"nogizaka46": 0, "sakurazaka46": 1, "hinatazaka46": 2}
	sort.Slice(result, func(i, j int) bool {
		gi, gj := groupOrder[result[i].Group], groupOrder[result[j].Group]
		if gi != gj {
			return gi < gj
		}
		if result[i].SingleNumber != result[j].SingleNumber {
			return result[i].SingleNumber < result[j].SingleNumber
		}
		return result[i].EventDate < result[j].EventDate
	})

	c.JSON(http.StatusOK, result)
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

func NormalizeMemberNames(c *gin.Context) {
	if !checkAdmin(c) {
		return
	}
	r1 := db.DB.Exec(`UPDATE records SET member_name = REPLACE(REPLACE(member_name, ' ', ''), '　', '') WHERE member_name LIKE '% %' OR member_name LIKE '%　%'`)
	r2 := db.DB.Exec(`UPDATE purchases SET member_name = REPLACE(REPLACE(member_name, ' ', ''), '　', ''), item_key = REPLACE(REPLACE(item_key, ' ', ''), '　', '') WHERE member_name LIKE '% %' OR member_name LIKE '%　%'`)
	r3 := db.DB.Exec(`UPDATE full_records SET member_name = REPLACE(REPLACE(member_name, ' ', ''), '　', '') WHERE member_name LIKE '% %' OR member_name LIKE '%　%'`)
	r4 := db.DB.Exec(`UPDATE sign_events SET member_name = REPLACE(REPLACE(member_name, ' ', ''), '　', '') WHERE member_name LIKE '% %' OR member_name LIKE '%　%'`)
	c.JSON(http.StatusOK, gin.H{
		"records":      r1.RowsAffected,
		"purchases":    r2.RowsAffected,
		"full_records": r3.RowsAffected,
		"sign_events":  r4.RowsAffected,
	})
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
