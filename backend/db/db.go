package db

import (
	"log"

	"fortune-tracker/config"
	"fortune-tracker/models"

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

	if err := DB.AutoMigrate(&models.User{}, &models.Record{}, &models.FullRecord{}); err != nil {
		log.Fatal("AutoMigrate failed:", err)
	}

	// 回填既有記錄的 order_id（冪等，只更新空值）
	DB.Exec(`
		UPDATE records
		SET order_id = SUBSTRING(source_url FROM '/apply_detail/([0-9]+)/')
		WHERE (order_id IS NULL OR order_id = '')
		  AND source_url LIKE '%/apply_detail/%'
	`)
}
