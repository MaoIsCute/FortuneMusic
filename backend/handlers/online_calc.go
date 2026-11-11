package handlers

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

// 付款模擬（/online-calc）不是長期紀錄，而是「剛開完第 N 抽、要決定付哪幾張單」時用的工具：
//   - 已確定 = forTUNE meets 券面頁的保有数（ticket_holdings 快照），已經是第 1～N-1 抽實際付過款的總和
//   - 本次   = 這張單曲還在「お支払い方法の未登録一覧」上的中選單（unpaid_applies 快照），一張一張讓使用者勾；
//             付過款的單已經在保有数裡，不能再勾（實測 43單：保有数 117 = 一抽中選 45 + 已付款的二抽中選 72）
// 所以二抽、三抽……都是同一套邏輯，不限定二抽。
//
// 券面還沒同步過（舊版擴充功能、或剛裝好還沒跑過個握同步）時，退回用「最新一抽之前所有抽次的
// 当選数」推估已確定張數，並回傳 has_ticket=false 讓前端提示——這個推估假設前幾抽全部都有付款。

var sessionDigits = regexp.MustCompile(`\d+`)

// isPlaceholderTitle 判斷 single_name 是不是「還沒公布標題」的佔位值（空白或含 タイトル未定）。
func isPlaceholderTitle(name string) bool {
	return name == "" || strings.Contains(name, "タイトル未定")
}

// sessionOrder 取出「第N部」裡的 N 當排序鍵。直接用字串排會讓「第10部」排在「第2部」前面。
func sessionOrder(session string) int {
	m := sessionDigits.FindString(session)
	if m == "" {
		return 0
	}
	n, err := strconv.Atoi(m)
	if err != nil {
		return 0
	}
	return n
}

// onlineCalcSlot 是一個 (日期, 部, 成員) 格子的張數，對應券面頁表格的一列。
type onlineCalcSlot struct {
	EventDate  string `json:"event_date"`
	Session    string `json:"session"`
	TimeRange  string `json:"time_range,omitempty"`
	MemberName string `json:"member_name"`
	Count      int    `json:"count"`
}

// onlineCalcOrder 是一張可以勾選的最新一抽中選單（order_id 就是 /mypage/apply_detail/<id>/ 的 id）。
type onlineCalcOrder struct {
	OrderID      string           `json:"order_id"`
	LotteryRound int              `json:"lottery_round"`
	TotalCount   int              `json:"total_count"`
	Slots        []onlineCalcSlot `json:"slots"`
}

type onlineCalcSummary struct {
	Group           string            `json:"group"`
	SingleNumber    int               `json:"single_number"`
	SingleName      string            `json:"single_name"`
	LatestRound     int               `json:"latest_round"`
	HasTicket       bool              `json:"has_ticket"`
	TicketScrapedAt *time.Time        `json:"ticket_scraped_at"`
	Base            []onlineCalcSlot  `json:"base"`
	BaseTotal       int               `json:"base_total"`
	Orders          []onlineCalcOrder `json:"orders"`
	// PaidOrderCount 是最新一抽裡已經付過款（不在未付款清單上）、所以已經算進券面保有数的中選單張數，
	// 只在 has_ticket 時有意義，前端拿來提示「第N抽有幾張單已包含在已確定裡」
	PaidOrderCount int `json:"paid_order_count"`
}

type slotKey struct {
	eventDate  string
	session    string
	memberName string
}

// sortSlots 依 日期 → 部 → 成員 排序，跟券面頁的呈現順序一致。
func sortSlots(slots []onlineCalcSlot) {
	sort.Slice(slots, func(i, j int) bool {
		a, b := slots[i], slots[j]
		if a.EventDate != b.EventDate {
			return a.EventDate < b.EventDate
		}
		if oa, ob := sessionOrder(a.Session), sessionOrder(b.Session); oa != ob {
			return oa < ob
		}
		if a.Session != b.Session {
			return a.Session < b.Session
		}
		return a.MemberName < b.MemberName
	})
}

func collapseSlots(agg map[slotKey]int) []onlineCalcSlot {
	slots := make([]onlineCalcSlot, 0, len(agg))
	for k, count := range agg {
		slots = append(slots, onlineCalcSlot{EventDate: k.eventDate, Session: k.session, MemberName: k.memberName, Count: count})
	}
	sortSlots(slots)
	return slots
}

// GetOnlineCalcSingles 回傳每個團體「最新的那張單曲」（單曲號最大，來源是有中選的個握紀錄或券面快照）。
// 這頁只關心眼前這張單曲要付哪幾張，舊單曲不列；多個團體都有資料時才讓前端顯示團體切換，
// 依最晚的活動日期排序，最近還有活動的團體排第一個（前端預設選它）。
func GetOnlineCalcSingles(c *gin.Context) {
	userID := getUserID(c)

	type singleRow struct {
		Group        string `json:"group"`
		SingleNumber int    `json:"single_number"`
		SingleName   string `json:"single_name"`
		LastDate     string `json:"last_date"`
	}
	var rows []singleRow
	// single_name 優先取非「タイトル未定」的版本：直接 MAX() 會因為片假名碼位比數字高，
	// 反而固定挑中 タイトル未定 那一版
	db.DB.Raw(`
		WITH src AS (
			SELECT "group", single_number, single_name, event_date FROM records
			 WHERE user_id = ? AND won_count > 0 AND single_number > 0
			UNION ALL
			SELECT "group", single_number, single_name, event_date FROM ticket_holdings
			 WHERE user_id = ? AND single_number > 0
		), latest AS (
			SELECT "group", MAX(single_number) AS single_number FROM src GROUP BY "group"
		)
		SELECT s."group", s.single_number,
		       COALESCE(MAX(NULLIF(CASE WHEN s.single_name LIKE '%タイトル未定%' THEN '' ELSE s.single_name END, '')),
		                MAX(s.single_name)) AS single_name,
		       MAX(s.event_date) AS last_date
		  FROM src s JOIN latest l ON l."group" = s."group" AND l.single_number = s.single_number
		 GROUP BY s."group", s.single_number
		 ORDER BY last_date DESC`, userID, userID).Scan(&rows)

	c.JSON(http.StatusOK, rows)
}

// GetOnlineCalcSummary 回傳一張單曲的模擬素材：券面保有数（已確定）＋最新一抽每張中選單的格子明細。
// 加總刻意留給前端做——勾選是純前端互動，每次勾都打後端只會讓畫面變鈍，資料量也很小。
func GetOnlineCalcSummary(c *gin.Context) {
	userID := getUserID(c)

	group := c.Query("group")
	singleNumber, err := strconv.Atoi(c.Query("single_number"))
	if group == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "請提供 group 與 single_number"})
		return
	}

	summary := onlineCalcSummary{
		Group:        group,
		SingleNumber: singleNumber,
		Base:         []onlineCalcSlot{},
		Orders:       []onlineCalcOrder{},
	}

	var records []models.Record
	db.DB.Where(`user_id = ? AND "group" = ? AND single_number = ? AND won_count > 0`, userID, group, singleNumber).
		Find(&records)
	for _, r := range records {
		if r.LotteryRound > summary.LatestRound {
			summary.LatestRound = r.LotteryRound
		}
		if summary.SingleName == "" || (isPlaceholderTitle(summary.SingleName) && !isPlaceholderTitle(r.SingleName)) {
			if r.SingleName != "" {
				summary.SingleName = r.SingleName
			}
		}
	}

	var holdings []models.TicketHolding
	db.DB.Where(`user_id = ? AND "group" = ? AND single_number = ?`, userID, group, singleNumber).Find(&holdings)

	if len(holdings) > 0 {
		summary.HasTicket = true
		at := holdings[0].ScrapedAt
		summary.TicketScrapedAt = &at
		for _, h := range holdings {
			if h.HeldCount <= 0 {
				continue
			}
			summary.Base = append(summary.Base, onlineCalcSlot{
				EventDate: h.EventDate, Session: h.Session, TimeRange: h.TimeRange,
				MemberName: h.MemberName, Count: h.HeldCount,
			})
			if summary.SingleName == "" || isPlaceholderTitle(summary.SingleName) {
				summary.SingleName = h.SingleName
			}
		}
		sortSlots(summary.Base)
	} else {
		baseAgg := map[slotKey]int{}
		for _, r := range records {
			if r.LotteryRound < summary.LatestRound {
				baseAgg[slotKey{r.EventDate, r.Session, r.MemberName}] += r.WonCount
			}
		}
		summary.Base = collapseSlots(baseAgg)
	}
	for _, s := range summary.Base {
		summary.BaseTotal += s.Count
	}

	// 哪些單可以勾：
	//   - 有券面：付過款的單已經在保有数裡，只能勾還在「未付款清單」上的中選單（不限抽次——
	//     前幾抽沒付、還沒過期的單也還能付）
	//   - 沒券面：已確定是用「最新一抽之前」的当選数推估的，可勾的就是最新一抽的全部中選單
	var unpaidSeqs []string
	if summary.HasTicket {
		db.DB.Model(&models.UnpaidApply{}).Where("user_id = ?", userID).Pluck("apply_seq", &unpaidSeqs)
	}
	isUnpaid := func(orderID string) bool {
		for _, seq := range unpaidSeqs {
			if strings.HasSuffix(orderID, seq) {
				return true
			}
		}
		return false
	}

	orderAgg := map[string]map[slotKey]int{}
	orderRound := map[string]int{}
	paidLatest := map[string]bool{}
	for _, r := range records {
		if summary.HasTicket {
			if !isUnpaid(r.OrderID) {
				if r.LotteryRound == summary.LatestRound {
					paidLatest[r.OrderID] = true
				}
				continue
			}
		} else if r.LotteryRound != summary.LatestRound {
			continue
		}
		if _, ok := orderAgg[r.OrderID]; !ok {
			orderAgg[r.OrderID] = map[slotKey]int{}
		}
		orderAgg[r.OrderID][slotKey{r.EventDate, r.Session, r.MemberName}] += r.WonCount
		orderRound[r.OrderID] = r.LotteryRound
	}
	summary.PaidOrderCount = len(paidLatest)
	for orderID, agg := range orderAgg {
		order := onlineCalcOrder{OrderID: orderID, LotteryRound: orderRound[orderID], Slots: collapseSlots(agg)}
		for _, s := range order.Slots {
			order.TotalCount += s.Count
		}
		summary.Orders = append(summary.Orders, order)
	}
	// 抽次新的在前，同抽次內單號大的（較新的應募）在前，跟網站抽選履歴的順序一致
	sort.Slice(summary.Orders, func(i, j int) bool {
		a, b := summary.Orders[i], summary.Orders[j]
		if a.LotteryRound != b.LotteryRound {
			return a.LotteryRound > b.LotteryRound
		}
		return a.OrderID > b.OrderID
	})

	c.JSON(http.StatusOK, summary)
}
