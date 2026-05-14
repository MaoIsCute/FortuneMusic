package router

import (
	"fortune-tracker/config"
	"fortune-tracker/handlers"
	"fortune-tracker/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(cfg *config.Config) *gin.Engine {
	r := gin.Default()

	r.GET("/auth/google", handlers.GoogleLogin(cfg))
	r.GET("/auth/google/callback", handlers.GoogleCallback(cfg))

	api := r.Group("/api", middleware.AuthRequired(cfg))
	{
		api.POST("/scrape", handlers.TriggerScrape)
		api.GET("/records", handlers.GetRecords)
		api.GET("/stats", handlers.GetStats)
	}

	return r
}
