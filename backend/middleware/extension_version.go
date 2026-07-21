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
const MinExtensionVersion = "2.1"

// LatestExtensionVersion 是目前擴充功能的最新版本。原本「有新版本」的提醒只做在網站
// /scrape 頁（見 CLAUDE.md #97），但使用者連結過帳號後幾乎不會再進那個頁面，只會直接在
// 擴充功能 popup 上點同步，提醒形同虛設。改成每次 /scrape/* 回應都帶上這個版本號，
// popup.js 的 backendFetch() 統一比對並顯示提醒，才會在使用者實際操作的路徑上被看到。
// 每次 bump manifest.json 版本號時，這裡要跟 status.json 的 latest_extension_version
// 一起改成同一個數字。
const LatestExtensionVersion = "2.4"

// ExtensionVersionRequired 擋掉版本過舊（或完全沒帶版本號，代表是這個機制上線前的舊擴充功能）的同步請求，
// 避免舊版持續送進已知有問題的資料，需要事後再花力氣 migration 修正；同時在每個回應都附上目前最新版本號
// （不管有沒有被擋下），供擴充功能自己比對是否落後
func ExtensionVersionRequired() gin.HandlerFunc {
	minVersion, _ := strconv.ParseFloat(MinExtensionVersion, 64)
	return func(c *gin.Context) {
		c.Header("X-Latest-Extension-Version", LatestExtensionVersion)
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
