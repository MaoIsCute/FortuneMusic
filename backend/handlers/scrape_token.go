package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

func GetScrapeToken(c *gin.Context) {
	userID := getUserID(c)
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if user.ScrapeToken == "" {
		b := make([]byte, 24)
		rand.Read(b)
		user.ScrapeToken = hex.EncodeToString(b)
		db.DB.Save(&user)
	}
	c.JSON(http.StatusOK, gin.H{"scrape_token": user.ScrapeToken})
}
