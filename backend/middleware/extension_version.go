package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// MinExtensionVersion 是後端接受同步請求的最低擴充功能版本。
// 每次提高門檻時，跟 extension/manifest.json 的 version、status.json 的
// latest_extension_version 一起在同一個 commit 裡改（見 CLAUDE.md「Git 操作規則」），
// 這樣三個版本號才不會漏改其中一個。
const MinExtensionVersion = "2.0"

// ExtensionVersionRequired 擋掉版本過舊（或完全沒帶版本號，代表是這個機制上線前的舊擴充功能）的同步請求，
// 避免舊版持續送進已知有問題的資料，需要事後再花力氣 migration 修正
func ExtensionVersionRequired() gin.HandlerFunc {
	minVersion, _ := strconv.ParseFloat(MinExtensionVersion, 64)
	return func(c *gin.Context) {
		version, err := strconv.ParseFloat(c.GetHeader("X-Extension-Version"), 64)
		if err != nil || version < minVersion {
			c.AbortWithStatusJSON(http.StatusUpgradeRequired, gin.H{
				"error":                 "extension_outdated",
				"message":               "同步工具版本過舊，請重新下載安裝最新版後再試",
				"min_extension_version": MinExtensionVersion,
			})
			return
		}
		c.Next()
	}
}
