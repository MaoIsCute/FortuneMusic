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

	// 將 lottery_round 從字串（"第N次"）轉換為整數（冪等，只在欄位型別為 varchar 時執行）
	for _, tbl := range []string{"records", "purchases"} {
		DB.Exec(`
			DO $$
			BEGIN
				IF EXISTS (
					SELECT 1 FROM information_schema.columns
					WHERE table_name = '` + tbl + `' AND column_name = 'lottery_round'
					  AND data_type IN ('character varying','text')
				) THEN
					ALTER TABLE ` + tbl + ` ALTER COLUMN lottery_round TYPE integer
					USING COALESCE((regexp_match(lottery_round, '(\d+)'))[1]::integer, 0);
				END IF;
			END $$;
		`)
	}

	if err := DB.AutoMigrate(&models.User{}, &models.Record{}, &models.FullRecord{}, &models.Purchase{}, &models.ScrapeLog{}, &models.TitleCorrection{}); err != nil {
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
