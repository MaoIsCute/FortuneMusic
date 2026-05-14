package handlers

import (
	"net/http"

	"fortune-tracker/db"
	"fortune-tracker/scraper"

	"github.com/gin-gonic/gin"
)

func TriggerScrape(c *gin.Context) {
	userID, _ := c.Get("userID")
	uid := userID.(uint)

	records, err := scraper.Scrape(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(records) > 0 {
		db.DB.Create(&records)
	}
	c.JSON(http.StatusOK, gin.H{"scraped": len(records), "records": records})
}
