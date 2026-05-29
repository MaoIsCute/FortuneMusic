package main

import (
	"os"

	"fortune-tracker/config"
	"fortune-tracker/db"
	"fortune-tracker/router"
)

func main() {
	cfg := config.Load()
	db.Init(cfg)
	r := router.Setup(cfg)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
