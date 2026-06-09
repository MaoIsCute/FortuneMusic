package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func ImpersonateMiddleware(adminEmail string) gin.HandlerFunc {
	return func(c *gin.Context) {
		impersonateID := c.GetHeader("X-Impersonate-User")
		if impersonateID == "" {
			c.Next()
			return
		}
		email, _ := c.Get("email")
		if adminEmail == "" || email != adminEmail {
			c.Next()
			return
		}
		uid, err := strconv.ParseUint(impersonateID, 10, 64)
		if err != nil {
			c.Next()
			return
		}
		c.Set("userID", uint(uid))
		c.Next()
	}
}
