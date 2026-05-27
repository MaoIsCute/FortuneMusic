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
	email, _ := c.Get("email")
	isAdmin := configuredAdminEmail != "" && email == configuredAdminEmail
	c.JSON(http.StatusOK, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"name":       user.Name,
		"is_admin":   isAdmin,
		"created_at": user.CreatedAt,
	})
}
