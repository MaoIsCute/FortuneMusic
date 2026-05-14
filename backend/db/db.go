package db

import (
    "log"
    "fortune-tracker/config"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

var DB *gorm.DB

func Init(cfg *config.Config) {
    var err error
    DB, err = gorm.Open(postgres.Open(cfg.DBUrl), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    log.Println("Database connected")
}
