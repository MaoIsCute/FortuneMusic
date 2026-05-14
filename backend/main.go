package main

import (
	"fortune-tracker/config"
	"fortune-tracker/db"
	"fortune-tracker/models"
	"fortune-tracker/router"
)

func main() {
	cfg := config.Load()
	db.Init(cfg)
	db.DB.AutoMigrate(&models.User{}, &models.Record{})
	r := router.Setup(cfg)
	r.Run(":8080")
}
