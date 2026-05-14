package scraper

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"fortune-tracker/models"

	"github.com/PuerkitoBio/goquery"
)

// targetURL is the page listing lottery/fortune results to scrape.
// Replace with the actual target and adjust selectors below.
const targetURL = "https://example.com/fortune-results"

func Scrape(userID uint) ([]models.Record, error) {
	resp, err := http.Get(targetURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var records []models.Record
	now := time.Now()

	// Adjust the selector to match the actual page structure.
	doc.Find("table.result-table tbody tr").Each(func(i int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 6 {
			return
		}
		records = append(records, models.Record{
			UserID:       userID,
			EventName:    clean(cells.Eq(0).Text()),
			MemberName:   clean(cells.Eq(1).Text()),
			EventDate:    clean(cells.Eq(2).Text()),
			Session:      clean(cells.Eq(3).Text()),
			AppliedCount: parseInt(cells.Eq(4).Text()),
			WonCount:     parseInt(cells.Eq(5).Text()),
			SourceURL:    targetURL,
			ScrapedAt:    now,
		})
	})
	return records, nil
}

func clean(s string) string {
	return strings.TrimSpace(s)
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n
}
