# Fortune Tracker — Project 說明文件

## 專案概述
統計 **Fortune Music** 網站上乃木坂46見面會抽選結果的工具。
使用者登入後可觸發爬蟲抓取自己的抽選紀錄，並在統計畫面分析中選機率。

---

## 技術選型

| 層級 | 技術 |
|------|------|
| 前端 | Vue 3 + Vue Router + Pinia + Element Plus |
| 後端 | Go + Gin + GORM |
| 資料庫 | PostgreSQL（Supabase 托管，免費方案） |
| 登入 | Google OAuth 2.0 + JWT |
| 部署 | Fly.io（免費方案） |
| 備份 | Supabase 內建每日自動備份 |

---

## 專案結構

```
D:\Project\fortunemusic\
├── backend\
│   ├── main.go
│   ├── .env
│   ├── config/
│   │   └── config.go           # 環境變數設定
│   ├── db/
│   │   └── db.go               # PostgreSQL 連線
│   ├── models/
│   │   ├── user.go             # User 資料表
│   │   └── record.go           # Record 資料表
│   ├── handlers/
│   │   ├── auth.go             # Google OAuth 登入
│   │   ├── scraper.go          # 爬蟲觸發 API
│   │   └── stats.go            # 統計 API
│   ├── scraper/
│   │   └── scraper.go          # goquery 爬蟲邏輯
│   ├── middleware/
│   │   └── auth.go             # JWT 驗證 middleware
│   └── router/
│       └── router.go           # 路由設定
└── frontend\
    ├── .env
    └── src/
        ├── main.js
        ├── App.vue
        ├── api/
        │   └── index.js        # Axios API 呼叫
        ├── router/
        │   └── index.js        # Vue Router 路由設定
        ├── stores/
        │   ├── auth.js         # 登入狀態管理
        │   └── theme.js        # 主題/應援色管理
        ├── styles/
        │   └── theme.js        # 成員應援色設定
        ├── components/
        │   └── NavBar.vue      # 導覽列
        └── views/
            ├── LoginView.vue   # 登入頁
            ├── DashboardView.vue # 統計總覽頁
            ├── MemberView.vue  # 成員統計頁
            ├── RecordsView.vue # 抽選紀錄列表頁
            └── ScrapeView.vue  # 爬蟲觸發頁
```

---

## 資料庫設計

### users 資料表
| 欄位 | 型別 | 說明 |
|------|------|------|
| id | INTEGER | 主鍵，自動遞增 |
| google_id | TEXT | Google 帳號 ID（唯一） |
| email | TEXT | Email（唯一） |
| name | TEXT | 顯示名稱 |
| created_at | DATETIME | 建立時間 |

### records 資料表
| 欄位 | 型別 | 說明 |
|------|------|------|
| id | INTEGER | 主鍵，自動遞增 |
| user_id | INTEGER | 關聯 users.id |
| event_name | TEXT | 活動名稱 |
| member_name | TEXT | 成員名稱 |
| event_date | TEXT | 活動日期（例：4/19） |
| session | TEXT | 部數（例：第2部） |
| applied_count | INTEGER | 應募數 |
| won_count | INTEGER | 中選數 |
| source_url | TEXT | 來源網址 |
| scraped_at | DATETIME | 爬取時間 |

---

## API 端點

### 認證
| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | /auth/google | 導向 Google OAuth |
| GET | /auth/google/callback | Google OAuth callback |

### 爬蟲
| 方法 | 路徑 | 說明 |
|------|------|------|
| POST | /api/scrape | 手動觸發爬蟲（需帶 Cookie） |

### 統計
| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | /api/stats/overall | 總中選率 |
| GET | /api/stats/by-date | 每日中選率 |
| GET | /api/stats/by-session | 每部中選率 |
| GET | /api/stats/by-member | 每個成員統計 |
| GET | /api/records | 所有抽選紀錄 |

---

## 爬蟲邏輯

**目標網站：** `fortunemusic.jp`（需登入）

**爬取流程：**
1. 使用者提供 Cookie → 後端帶 Cookie 請求
2. 爬取 `fortunemusic.jp/mypage/apply_list/` → 取得所有訂單連結
3. 逐一爬取 `fortunemusic.jp/mypage/apply_detail/{ID}/`
4. 解析商品名稱，提取成員名、日期、部數
5. 存入資料庫（已存在的 ID 跳過，避免重複）

**商品名稱格式：**
```
{成員名}【{月/日} {第N部}】{活動名稱}
```
範例：`奥田いろは【4/19 第2部】乃木坂46 41stシングル...`

---

## 前端頁面

| 路徑 | 頁面 | 說明 |
|------|------|------|
| / | LoginView | Google 登入頁 |
| /dashboard | DashboardView | 統計總覽、成員列表 |
| /member/:name | MemberView | 成員詳細統計（日期別／部數別／紀錄） |
| /records | RecordsView | 所有紀錄列表，可依成員篩選 |
| /scrape | ScrapeView | 觸發爬蟲，Cookie 輸入 |

---

## 主題系統

選擇成員時，全站配色跟著切換為該成員的應援色漸層。
支援深色／淺色模式切換。

**目前設定的成員應援色：**
| 成員 | 主色 | 副色 |
|------|------|------|
| 五百城茉央 | #40E0D0（ターコイズ） | #1E90FF（青） |

---

## 環境變數

### backend/.env
```
DATABASE_URL=postgresql://postgres:PASSWORD@db.SUPABASE_ID.supabase.co:5432/postgres
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret
JWT_SECRET=your_random_secret_string
APP_URL=http://localhost:8080
```

### frontend/.env
```
VITE_API_URL=http://localhost:8080
```

---

## 待完成項目

- [ ] 後端 Google OAuth 完整實作（handlers/auth.go）
- [ ] 後端 JWT middleware 實作（middleware/auth.go）
- [ ] 後端爬蟲邏輯實作（scraper/scraper.go）
- [ ] 後端統計 API 實作（handlers/stats.go）
- [ ] 後端路由設定（router/router.go）
- [ ] Google Cloud Console 建立 OAuth 憑證
- [ ] Supabase 建立專案取得 DATABASE_URL
- [ ] 前後端串接測試
- [ ] Fly.io 部署設定
- [ ] 新增更多成員應援色