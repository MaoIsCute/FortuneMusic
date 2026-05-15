# Fortune Tracker — 專案說明

乃木坂46 抽選統計工具。使用者以 Google 帳號登入，透過瀏覽器擴充功能自動抓取 Fortune Music 的抽選記錄並統計。

## Tech Stack

| 層 | 技術 |
|---|---|
| 前端 | Vue 3 + Vite + Element Plus + Pinia |
| 後端 | Go + Gin |
| 資料庫 | PostgreSQL（Supabase） |
| 驗證 | Google OAuth 2.0 + JWT（15 分鐘過期） |

## 專案結構

```
fortunemusic/
├── backend/
│   ├── main.go
│   ├── config/config.go          # 讀取 .env
│   ├── db/db.go                  # Gorm + PostgreSQL
│   ├── models/
│   │   ├── user.go               # User{ID, GoogleID, Email, Name, ScrapeToken}
│   │   └── record.go             # 抽選記錄
│   ├── middleware/auth.go        # JWT 驗證
│   ├── handlers/
│   │   ├── auth.go               # Google OAuth callback → JWT
│   │   ├── user.go               # GET /api/me
│   │   ├── scraper.go            # TriggerScrape（JWT）、PublicScrape（ScrapeToken）
│   │   ├── scrape_token.go       # GET /api/scrape-token（生成/取得 ScrapeToken）
│   │   └── stats.go              # 各種統計 endpoint
│   ├── scraper/scraper.go        # 爬蟲邏輯（goquery）
│   └── router/router.go          # 路由 + CORS
├── frontend/
│   ├── src/
│   │   ├── api/index.js          # axios + interceptors
│   │   ├── stores/auth.js        # Pinia auth store（token 存 localStorage）
│   │   ├── router/index.js       # Vue Router（含未登入路由保護）
│   │   └── views/
│   │       ├── LoginView.vue
│   │       ├── AuthCallbackView.vue
│   │       ├── DashboardView.vue
│   │       ├── MemberView.vue
│   │       ├── RecordsView.vue
│   │       └── ScrapeView.vue    # 顯示 ScrapeToken + 手動貼 Cookie
│   └── vite.config.js            # proxy: /auth/google → localhost:8080
└── extension/                    # Chrome 擴充功能
    ├── manifest.json             # Manifest V3
    ├── popup.html
    ├── popup.js
    └── popup.css
```

## 路由一覽

### Backend

| 方法 | 路徑 | 說明 |
|---|---|---|
| GET | `/auth/google` | 導向 Google OAuth |
| GET | `/auth/google/callback` | Google 回調，產生 JWT，redirect 到前端 |
| POST | `/scrape` | 公開端點，用 ScrapeToken 驗證（擴充功能用） |
| GET | `/api/me` | 取得登入使用者資訊 |
| GET | `/api/scrape-token` | 取得/生成 ScrapeToken |
| POST | `/api/scrape` | 手動貼 Cookie 觸發爬蟲（JWT 驗證） |
| GET | `/api/stats/overall` | 整體統計 |
| GET | `/api/stats/by-member` | 依成員統計 |
| GET | `/api/stats/by-date` | 依日期統計 |
| GET | `/api/stats/by-session` | 依場次統計 |
| GET | `/api/records` | 取得記錄（page_size 上限 100） |

### Frontend

| 路徑 | 說明 |
|---|---|
| `/` | 登入頁 |
| `/auth/callback` | Google OAuth 回調頁（存 token → redirect） |
| `/dashboard` | 主畫面（統計總覽） |
| `/member/:name` | 個別成員統計 |
| `/records` | 記錄列表 |
| `/scrape` | 爬蟲頁（ScrapeToken + 手動 Cookie） |

## 驗證流程

```
LoginView → GET http://localhost:8080/auth/google
  → Google OAuth
  → GET /auth/google/callback（後端）
  → JWT 產生 → redirect 前端 /auth/callback?token=...
  → AuthCallbackView 存 token → redirect 到原始目標路由（localStorage）
```

**注意：** Vite proxy 只代理 `/auth/google`，不代理 `/auth/callback`（否則會 404）。

## ScrapeToken 機制

- `User.ScrapeToken` 存在資料庫，長期有效（不像 JWT 15 分鐘過期）
- 使用者在 ScrapeView 點「取得 Token」生成一次，複製到擴充功能
- 擴充功能用此 token 呼叫公開的 `POST /scrape`，不需要 JWT

## Chrome 擴充功能

安裝方式：`chrome://extensions/` → 開發人員模式 → 載入未封裝項目 → 選 `extension/` 資料夾

首次設定：
1. 輸入後端網址（預設 `http://localhost:8080`）
2. 從 ScrapeView 複製 ScrapeToken 貼上 → 儲存

使用方式：在 `fortunemusic.jp` 登入後點擴充功能圖示 → 「同步」

擴充功能會同時抓取 `fortunemusic.jp` 和 `main.fortunemusic.jp` 兩個子網域的 cookie。

## 爬蟲邏輯（待確認）

目標網站：`https://fortunemusic.jp`

```
1. GET /mypage/apply_list/（帶 Cookie）
2. 掃描頁面所有 <a href>，找符合 /mypage/apply_detail/(\d+)/ 的連結
3. 對每個訂單 ID，GET /mypage/apply_detail/{id}/
4. 找 .apply-item / .order-item / tr.item-row 元素
5. 解析商品名稱 regex：成員名【月/日 第N部】活動名稱
```

### 目前已知問題（正在調查）

- `GET /mypage/apply_list/` 取回的是登入頁（title: "forTUNE music"，有登入表單）
- 頁面連結出現 `main.fortunemusic.jp`，代表會員系統可能在子網域
- 根因推測：使用者 session 在 `main.fortunemusic.jp`，`fortunemusic.jp` 的 cookie 無效；若同名 cookie 合併時 main 的 session 被蓋掉

### 已做的修改（待驗證）

- **擴充功能**：不再合併兩域 cookie，改分開傳 `cookie_fortune`（fortunemusic.jp）與 `cookie_main`（main.fortunemusic.jp）
- **後端 handler**：接受 `cookie_fortune` / `cookie_main`；舊 `cookie` 欄位向後相容
- **爬蟲**：先試 `fortunemusic.jp`；若是登入頁則改用 `main.fortunemusic.jp` + 對應 cookie 試相同路徑
- **debug 強化**：顯示各域 cookie 名稱清單、每個 domain 嘗試結果、最終跳轉 URL

## .env 設定（backend）

```
DATABASE_URL=...
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
JWT_SECRET=...
APP_URL=http://localhost:8080
FRONTEND_URL=http://localhost:5173
```

## 已修正的問題

1. Vite proxy `/auth` 太寬，改成 `/auth/google`（否則 `/auth/callback` 會被導到後端 → 404）
2. JWT 過期無聲失敗 → 加 401 interceptor，自動清 token 並提示「登入已過期」
3. `page_size: 500` 超過後端上限 100 被靜默忽略 → 改成 100
4. 登入後未保留原始目標路由 → 用 localStorage 暫存，登入後跳回
