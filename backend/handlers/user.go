package handlers

import (
	"net/http"

	"fortune-tracker/db"
	"fortune-tracker/models"

	"github.com/gin-gonic/gin"
)

func GetMe(c *gin.Context) {
	userID := getUserID(c)
	var user models.User
	if err := db.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}
