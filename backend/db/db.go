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

	// title_corrections 改名為 titles：原本只是「タイトル未定修正對照表」，現在當成主動維護的單曲名稱主表用（一次性，保留既有資料）
	DB.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'title_corrections')
			   AND NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'titles') THEN
				ALTER TABLE title_corrections RENAME TO titles;
			END IF;
		END $$;
	`)
	DB.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_title_correction_group_single')
			   AND NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_title_group_single') THEN
				ALTER INDEX idx_title_correction_group_single RENAME TO idx_title_group_single;
			END IF;
		END $$;
	`)

	// titles 補上 group 欄位，唯一鍵從 single_number 單欄改成 (group, single_number) 複合鍵（冪等，相容更早期沒有 group 欄位的資料）
	// 既有資料的 group 設為空字串：不會跟任何真實 group 撞鍵，等於需要透過 admin 重新登記
	DB.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'titles')
			   AND NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'titles' AND column_name = 'group'
			   ) THEN
				ALTER TABLE titles ADD COLUMN "group" varchar(255) NOT NULL DEFAULT '';
			END IF;
		END $$;
	`)
	DB.Exec(`
		DO $$
		DECLARE idx_name text;
		BEGIN
			SELECT indexname INTO idx_name FROM pg_indexes
			WHERE tablename = 'titles' AND indexdef LIKE '%UNIQUE%' AND indexdef LIKE '%(single_number)%';
			IF idx_name IS NOT NULL THEN
				EXECUTE 'DROP INDEX IF EXISTS ' || quote_ident(idx_name);
			END IF;
		END $$;
	`)

	// titles 新增 org_album_name 欄位供專輯改名對照（冪等），並更新 unique index 加入第三欄
	DB.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'titles')
			   AND NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'titles' AND column_name = 'org_album_name'
			   ) THEN
				ALTER TABLE titles ADD COLUMN org_album_name varchar(255) NOT NULL DEFAULT '';
			END IF;
		END $$;
	`)
	DB.Exec(`
		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE indexname = 'idx_title_group_single'
				  AND indexdef NOT LIKE '%org_album_name%'
			) THEN
				DROP INDEX idx_title_group_single;
			END IF;
		END $$;
	`)

	// titles 新增 release_date 欄位（冪等）
	DB.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'titles')
			   AND NOT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'titles' AND column_name = 'release_date'
			   ) THEN
				ALTER TABLE titles ADD COLUMN release_date date;
			END IF;
		END $$;
	`)

	if err := DB.AutoMigrate(&models.User{}, &models.Record{}, &models.FullRecord{}, &models.SignEvent{}, &models.Purchase{}, &models.ScrapeLog{}, &models.Title{}, &models.Venue{}); err != nil {
		log.Fatal("AutoMigrate failed:", err)
	}

	// 回填既有記錄的 order_id（冪等，只更新空值）
	DB.Exec(`
		UPDATE records
		SET order_id = SUBSTRING(source_url FROM '/apply_detail/([0-9]+)/')
		WHERE (order_id IS NULL OR order_id = '')
		  AND source_url LIKE '%/apply_detail/%'
	`)

	// 個握/花費的 session 統一補上「第」前綴（冪等）：來源網站對場次的顯示方式不一致，
	// 早期擴充功能解析時「第」是可有可無的（正規表達式 第?），導致「第1部」跟「1部」
	// 被當成兩個不同場次分開統計；擴充功能端已改成一律補「第」，這裡回填既有資料
	for _, tbl := range []string{"records", "purchases"} {
		DB.Exec(`UPDATE ` + tbl + ` SET session = '第' || session WHERE session ~ '^[0-9]+部$'`)
	}
}
