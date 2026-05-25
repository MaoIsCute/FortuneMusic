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
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			return origin == "http://localhost:5173" ||
				strings.HasPrefix(origin, "chrome-extension://")
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.GET("/auth/google", handlers.GoogleLogin(cfg))
	r.GET("/auth/google/callback", handlers.GoogleCallback(cfg))

	// 供瀏覽器擴充功能使用（以 scrape_token 驗證，不需 JWT）
	r.POST("/scrape", handlers.PublicScrape)
	r.POST("/scrape/push", handlers.PushRecords) // 擴充功能在瀏覽器內爬好後直接推送記錄

	api := r.Group("/api", middleware.AuthRequired(cfg))
	{
		api.GET("/me", handlers.GetMe)
		api.GET("/scrape-token", handlers.GetScrapeToken)
		api.POST("/scrape", handlers.TriggerScrape)
		api.GET("/records", handlers.GetRecords)
		api.GET("/stats", handlers.GetStats)
		api.GET("/stats/overall", handlers.GetOverallStats)
		api.GET("/stats/by-member", handlers.GetStatsByMember)
		api.GET("/stats/by-date", handlers.GetStatsByDate)
		api.GET("/stats/by-session", handlers.GetStatsBySession)
		api.GET("/stats/detail", handlers.GetDetailStats)
	}

	return r
}
