package scraper

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"fortune-tracker/db"
	"fortune-tracker/models"
)

const (
	baseFortune = "https://fortunemusic.jp"
	baseMain    = "https://main.fortunemusic.jp"
	applyList   = "/mypage/apply_list/"
	applyDetail = "/mypage/apply_detail/"
)

type Result struct {
	NewRecords int    `json:"new_records"`
	Skipped    int    `json:"skipped"`
	Message    string `json:"message"`
	Debug      string `json:"debug,omitempty"`
}

type orderEntry struct {
	id         string
	applyYear  int
	applyMonth int
}

func Run(userID uint, cookieFortune, cookieMain string) (*Result, error) {
	// cookiejar 讓 client 自動依 domain 帶對的 cookie，
	// 也處理 SSO redirect 時跨 domain 換 cookie 的問題。
	client := buildClient(cookieFortune, cookieMain)
	var dbg []string

	dbg = append(dbg, fmt.Sprintf("fortune cookies: [%s]", strings.Join(cookieNames(cookieFortune), ",")))
	if cookieMain != "" {
		dbg = append(dbg, fmt.Sprintf("main cookies: [%s]", strings.Join(cookieNames(cookieMain), ",")))
	}

	// 先試 fortunemusic.jp（jar 會在 redirect 時自動帶 main 的 cookie）
	entries, pageTitle, hrefs, err := fetchOrderIDs(client, baseFortune+applyList)
	dbg = append(dbg, fmt.Sprintf("[fortune] title=%q orders=%d login=%v err=%v", pageTitle, len(entries), isLoginPage(pageTitle), err))

	usedBase := baseFortune

	// 若還是失敗，直接試 main.fortunemusic.jp 幾個可能路徑
	if (err != nil || isLoginPage(pageTitle) || len(entries) == 0) && cookieMain != "" {
		for _, candidate := range []string{
			baseMain + applyList,
			baseMain + "/secure/apply/applyList.php?site=F",
			baseMain + "/secure/ticket/applyList.php?site=F",
		} {
			entriesM, titleM, hrefsM, errM := fetchOrderIDs(client, candidate)
			dbg = append(dbg, fmt.Sprintf("[main] url=%s title=%q orders=%d login=%v err=%v", candidate, titleM, len(entriesM), isLoginPage(titleM), errM))
			if !isLoginPage(titleM) && len(hrefsM) > 0 {
				dbg = append(dbg, "[main] links: "+hrefsM)
			}
			if errM == nil && len(entriesM) > 0 {
				entries = entriesM
				hrefs = hrefsM
				usedBase = baseMain
				break
			}
		}
	}

	result := &Result{}
	result.Debug = strings.Join(dbg, " ｜ ")
	if len(entries) == 0 && hrefs != "" {
		result.Debug += " ｜ links: " + hrefs
	}

	for _, entry := range entries {
		detailURL := fmt.Sprintf("%s%s%s/", usedBase, applyDetail, entry.id)
		records, err := fetchOrderDetail(client, detailURL, entry.applyYear, entry.applyMonth)
		if err != nil {
			fmt.Printf("[scraper] 訂單 %s 爬取失敗: %v\n", entry.id, err)
			continue
		}

		for _, rec := range records {
			rec.UserID = userID
			rec.ScrapedAt = time.Now()

			var existing models.Record
			if db.DB.Where("user_id = ? AND source_url = ?", userID, rec.SourceURL).First(&existing).Error == nil {
				result.Skipped++
				continue
			}

			if err := db.DB.Create(&rec).Error; err != nil {
				fmt.Printf("[scraper] 寫入失敗: %v\n", err)
				continue
			}
			result.NewRecords++
		}
	}

	result.Message = fmt.Sprintf("完成！新增 %d 筆，跳過 %d 筆", result.NewRecords, result.Skipped)
	return result, nil
}

// buildClient 建立帶有 cookiejar 的 HTTP client，
// 讓 SSO redirect 跨 domain 時也能自動帶對的 cookie。
func buildClient(cookieFortune, cookieMain string) *http.Client {
	jar, _ := cookiejar.New(nil)

	if cookieFortune != "" {
		u, _ := url.Parse("https://fortunemusic.jp")
		jar.SetCookies(u, parseCookies(cookieFortune))
	}
	if cookieMain != "" {
		u, _ := url.Parse("https://main.fortunemusic.jp")
		jar.SetCookies(u, parseCookies(cookieMain))
	}

	return &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}
}

func parseCookies(cookieStr string) []*http.Cookie {
	var cookies []*http.Cookie
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		idx := strings.Index(part, "=")
		if idx <= 0 {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:  strings.TrimSpace(part[:idx]),
			Value: strings.TrimSpace(part[idx+1:]),
		})
	}
	return cookies
}

func fetchDoc(client *http.Client, targetURL string) (*goquery.Document, string, error) {
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL.String()
	if resp.StatusCode != http.StatusOK {
		return nil, finalURL, fmt.Errorf("HTTP %d: %s", resp.StatusCode, targetURL)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	return doc, finalURL, err
}

func fetchOrderIDs(client *http.Client, targetURL string) ([]orderEntry, string, string, error) {
	doc, finalURL, err := fetchDoc(client, targetURL)
	if err != nil {
		return nil, "", "", err
	}

	pageTitle := strings.TrimSpace(doc.Find("title").Text())
	pageTitle = fmt.Sprintf("%s (→%s)", pageTitle, finalURL)

	var entries []orderEntry
	seen := map[string]bool{}
	linkRe := regexp.MustCompile(`/mypage/apply_detail/(\d+)/?`)
	dateRe := regexp.MustCompile(`(\d{4})-(\d{1,2})-\d{1,2}`)

	var sampleHrefs []string
	doc.Find("a[href]").Each(func(_ int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		if len(sampleHrefs) < 20 && href != "" && href != "#" {
			sampleHrefs = append(sampleHrefs, href)
		}
		m := linkRe.FindStringSubmatch(href)
		if len(m) < 2 || seen[m[1]] {
			return
		}
		seen[m[1]] = true

		entry := orderEntry{id: m[1]}

		// 從最近的祖先容器中找 応募日時
		container := a.Closest("tr, li, article, section")
		if container.Length() == 0 {
			container = a.Parent()
		}
		container.Find("span.hdg").Each(func(_ int, span *goquery.Selection) {
			if strings.TrimSpace(span.Text()) != "応募日時" {
				return
			}
			tdText := strings.TrimSpace(span.Parent().Text())
			dateStr := strings.TrimSpace(strings.Replace(tdText, strings.TrimSpace(span.Text()), "", 1))
			if dm := dateRe.FindStringSubmatch(dateStr); len(dm) >= 3 {
				entry.applyYear, _ = strconv.Atoi(dm[1])
				entry.applyMonth, _ = strconv.Atoi(dm[2])
			}
		})

		entries = append(entries, entry)
	})

	return entries, pageTitle, strings.Join(sampleHrefs, " | "), nil
}

// inferEventYear は應募月を基準に活動年分を決定する。
// 活動月 < 應募月 の場合は翌年（例：12月応募 → 1月活動 = 翌年）
func inferEventYear(rawDate string, applyYear, applyMonth int) int {
	if applyYear == 0 {
		return time.Now().Year()
	}
	parts := strings.SplitN(rawDate, "/", 2)
	eventMonth, err := strconv.Atoi(parts[0])
	if err != nil {
		return applyYear
	}
	if eventMonth < applyMonth {
		return applyYear + 1
	}
	return applyYear
}

func fetchOrderDetail(client *http.Client, targetURL string, applyYear, applyMonth int) ([]*models.Record, error) {
	doc, _, err := fetchDoc(client, targetURL)
	if err != nil {
		return nil, err
	}

	var records []*models.Record
	doc.Find(".apply-item, .order-item, tr.item-row").Each(func(_ int, s *goquery.Selection) {
		if rec := parseItem(s, targetURL, applyYear, applyMonth); rec != nil {
			records = append(records, rec)
		}
	})

	if len(records) == 0 {
		records = parseByText(doc, targetURL, applyYear, applyMonth)
	}

	return records, nil
}

func isLoginPage(title string) bool {
	t := strings.TrimSpace(title)
	if idx := strings.Index(t, " (→"); idx > 0 {
		t = t[:idx]
	}
	return t == "forTUNE music" ||
		strings.Contains(strings.ToLower(t), "login") ||
		strings.Contains(t, "ログイン")
}

func cookieNames(cookieStr string) []string {
	var names []string
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		if idx := strings.Index(part, "="); idx > 0 {
			names = append(names, part[:idx])
		}
	}
	return names
}

func parseItem(s *goquery.Selection, sourceURL string, applyYear, applyMonth int) *models.Record {
	productName := strings.TrimSpace(s.Find(".product-name, .item-name, td:first-child").First().Text())
	if productName == "" {
		productName = strings.TrimSpace(s.Text())
	}

	member, date, session, eventName := parseProductName(productName)
	if member == "" {
		return nil
	}

	won := 0
	if strings.Contains(s.Text(), "当選") {
		won = 1
	}

	appliedText := strings.TrimSpace(s.Find(".applied-count, td.count").First().Text())
	applied := 0
	if n, err := strconv.Atoi(regexp.MustCompile(`\d+`).FindString(appliedText)); err == nil {
		applied = n
	}

	eventYear := inferEventYear(date, applyYear, applyMonth)
	itemURL := fmt.Sprintf("%s#%s|%s|%s", sourceURL, url.QueryEscape(member), date, session)
	return &models.Record{
		EventName:    eventName,
		MemberName:   member,
		EventDate:    fmt.Sprintf("%d/%s", eventYear, date),
		Session:      session,
		AppliedCount: applied,
		WonCount:     won,
		SourceURL:    itemURL,
	}
}

func parseByText(doc *goquery.Document, sourceURL string, applyYear, applyMonth int) []*models.Record {
	var records []*models.Record
	seen := map[string]bool{}
	re := regexp.MustCompile(`[\p{Han}\p{Hiragana}\p{Katakana}a-zA-Z]+【\d{1,2}/\d{1,2}\s+第\d+部】.+`)

	doc.Find("*").Each(func(_ int, s *goquery.Selection) {
		text := strings.TrimSpace(s.Text())
		if re.MatchString(text) && len(text) < 200 {
			member, date, session, eventName := parseProductName(text)
			if member == "" {
				return
			}
			key := member + date + session
			if seen[key] {
				return
			}
			seen[key] = true
			eventYear := inferEventYear(date, applyYear, applyMonth)
			itemURL := fmt.Sprintf("%s#%s|%s|%s", sourceURL, url.QueryEscape(member), date, session)
			records = append(records, &models.Record{
				EventName:  eventName,
				MemberName: member,
				EventDate:  fmt.Sprintf("%d/%s", eventYear, date),
				Session:    session,
				SourceURL:  itemURL,
			})
		}
	})

	return records
}

func parseProductName(name string) (member, date, session, eventName string) {
	re := regexp.MustCompile(`^(.+?)【(\d{1,2}/\d{1,2})\s+(第\d+部)】(.+)$`)
	m := re.FindStringSubmatch(strings.TrimSpace(name))
	if len(m) != 5 {
		return
	}
	return strings.TrimSpace(m[1]), m[2], m[3], strings.TrimSpace(m[4])
}
