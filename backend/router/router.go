package router

import (
	"strings"

	"fortune-tracker/config"
	"fortune-tracker/handlers"
	"fortune-tracker/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Setup(cfg *config.Config) *gin.Engine {
	handlers.InitAdmin(cfg.AdminEmail)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://localhost:5173" ||
				(cfg.FrontendURL != "" && origin == cfg.FrontendURL) ||
				strings.HasPrefix(origin, "chrome-extension://")
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-Impersonate-User", "X-Extension-Version"},
		ExposeHeaders:    []string{"X-Latest-Extension-Version"},
		AllowCredentials: true,
	}))

	// Gin 不像 net/http 的 ServeMux 會自動把 HEAD 導到同路徑的 GET handler，兩個方法要分開註冊；
	// UptimeRobot 的 HTTP(s) monitor 預設是送 HEAD 請求，只註冊 GET 的話 HEAD 會直接 404（連
	// handler 都沒被呼叫到），導致監控其實沒有真的碰到後端、Render 免費方案還是會照樣休眠
	healthCheck := func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) }
	r.GET("/health", healthCheck)
	r.HEAD("/health", healthCheck)

	r.GET("/auth/google", handlers.GoogleLogin(cfg))
	r.GET("/auth/google/callback", handlers.GoogleCallback(cfg))
	r.POST("/auth/token", handlers.ExchangeToken)
	r.POST("/auth/refresh", handlers.RefreshJWT(cfg))

	// 供瀏覽器擴充功能使用（以 scrape_token 驗證，不需 JWT），額外要求 X-Extension-Version
	// header 達到最低版本才放行（見 middleware.ExtensionVersionRequired）
	scrape := r.Group("/scrape", middleware.ExtensionVersionRequired())
	{
		scrape.POST("", handlers.PublicScrape)
		scrape.POST("/push", handlers.PushRecords)
		scrape.POST("/check-orders", handlers.CheckOrders)
		scrape.POST("/update-titles", handlers.UpdateTitles)
		scrape.POST("/full/push", handlers.PushFullRecords)
		scrape.POST("/check-entries", handlers.CheckEntries)
		scrape.POST("/purchases/push", handlers.PushPurchases)
		scrape.POST("/log", handlers.PushScrapeLog)
	}

	api := r.Group("/api", middleware.AuthRequired(cfg), middleware.ImpersonateMiddleware(cfg.AdminEmail))
	{
		api.GET("/me", handlers.GetMe)
		api.GET("/admin/users", handlers.GetAdminUsers)
		api.GET("/admin/users/:id/records/preview", handlers.PreviewUserRecords)
		api.GET("/admin/users/:id/full-records/preview", handlers.PreviewUserFullRecords)
		api.GET("/admin/users/:id/purchases/preview", handlers.PreviewUserPurchases)
		api.GET("/admin/users/:id/sign-events/preview", handlers.PreviewUserSignEvents)
		api.GET("/admin/users/:id/prizes/preview", handlers.PreviewUserPrizes)
		api.DELETE("/admin/users/:id/records", handlers.DeleteUserRecords)
		api.DELETE("/admin/users/:id/full-records", handlers.DeleteUserFullRecords)
		api.DELETE("/admin/users/:id/purchases", handlers.DeleteUserPurchases)
		api.DELETE("/admin/users/:id/sign-events", handlers.DeleteUserSignEvents)
		api.DELETE("/admin/users/:id/prizes", handlers.DeleteUserPrizes)
		api.POST("/admin/normalize-member-names", handlers.NormalizeMemberNames)
		api.GET("/admin/title-issues", handlers.GetTitleIssues)
		api.GET("/admin/titles", handlers.GetKnownTitles)
		api.PUT("/admin/title", handlers.FixSingleTitle)
		api.POST("/admin/titles/bulk", handlers.BulkSetTitles)
		api.GET("/admin/venue-issues", handlers.GetVenueIssues)
		api.PUT("/admin/venue", handlers.FixVenue)
		api.POST("/admin/venues/bulk", handlers.BulkSetVenues)
		api.GET("/admin/venues", handlers.GetKnownVenues)
		api.GET("/admin/scrape-logs", handlers.GetAdminScrapeLogs)
		api.GET("/admin/sign-events", handlers.GetAdminSignEvents)
		api.GET("/admin/prizes", handlers.GetAdminPrizes)
		api.GET("/scrape-token", handlers.GetScrapeToken)
		api.POST("/scrape", handlers.TriggerScrape)
		api.GET("/records", handlers.GetRecords)
		api.GET("/stats", handlers.GetStats)
		api.GET("/stats/overall", handlers.GetOverallStats)
		api.GET("/stats/by-member", handlers.GetStatsByMember)
		api.GET("/stats/by-date", handlers.GetStatsByDate)
		api.GET("/stats/by-session", handlers.GetStatsBySession)
		api.GET("/stats/detail", handlers.GetDetailStats)
		api.GET("/stats/order-sequence", handlers.GetOrderSequenceStats)
		api.GET("/sign-events", handlers.GetSignEvents)
		api.GET("/prizes", handlers.GetPrizes)
		api.PUT("/prizes/:id/result", handlers.UpdatePrizeResult)
		api.GET("/full/records", handlers.GetFullRecords)
		api.GET("/full/stats/overall", handlers.GetFullOverallStats)
		api.GET("/full/stats/by-member", handlers.GetFullStatsByMember)
		api.GET("/full/stats/by-single", handlers.GetFullStatsBySingle)
		api.GET("/full/stats/detail", handlers.GetFullDetailStats)
		api.GET("/full/stats/by-region", handlers.GetFullStatsByRegion)
		api.GET("/purchases", handlers.GetPurchases)
		api.GET("/purchases/tree", handlers.GetPurchaseTree)
		api.GET("/purchases/stats/overall", handlers.GetPurchaseOverallStats)
		api.GET("/purchases/stats/by-single", handlers.GetPurchaseStatsBySingle)
		api.GET("/purchases/stats/by-member", handlers.GetPurchaseStatsByMember)
		api.GET("/global/stats/overall", handlers.GetGlobalOverallStats)
		api.GET("/global/stats/detail", handlers.GetGlobalDetailStats)
		api.GET("/global/stats/order-sequence", handlers.GetGlobalOrderSequenceStats)
		api.GET("/global/stats/by-member", handlers.GetGlobalStatsByMember)
		api.GET("/global/stats/by-single", handlers.GetGlobalStatsBySingle)
		api.GET("/global/full/stats/overall", handlers.GetGlobalFullOverallStats)
		api.GET("/global/full/stats/by-member", handlers.GetGlobalFullStatsByMember)
		api.GET("/global/full/stats/by-region", handlers.GetGlobalFullStatsByRegion)
		api.GET("/global/full/stats/detail", handlers.GetGlobalFullDetailStats)
	}

	return r
}
