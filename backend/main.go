package main

import (
	"fortune-tracker/config"
	"fortune-tracker/db"
	"fortune-tracker/router"
)

func main() {
	cfg := config.Load()
	db.Init(cfg)
	r := router.Setup(cfg)
	r.Run(":8080")
}
