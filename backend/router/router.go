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
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/auth/google", handlers.GoogleLogin(cfg))
	r.GET("/auth/google/callback", handlers.GoogleCallback(cfg))
	r.POST("/auth/token", handlers.ExchangeToken)
	r.POST("/auth/refresh", handlers.RefreshJWT(cfg))

	// 供瀏覽器擴充功能使用（以 scrape_token 驗證，不需 JWT）
	r.POST("/scrape", handlers.PublicScrape)
	r.POST("/scrape/push", handlers.PushRecords)
	r.POST("/scrape/check-orders", handlers.CheckOrders)
	r.POST("/scrape/update-titles", handlers.UpdateTitles)

	api := r.Group("/api", middleware.AuthRequired(cfg))
	{
		api.GET("/me", handlers.GetMe)
		api.GET("/admin/users", handlers.GetAdminUsers)
		api.DELETE("/admin/users/:id/records", handlers.DeleteUserRecords)
		api.GET("/admin/title-issues", handlers.GetTitleIssues)
		api.PUT("/admin/title", handlers.FixSingleTitle)
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
	}

	return r
}
