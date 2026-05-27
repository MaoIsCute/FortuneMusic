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
│   ├── middleware/auth.go        # JWT 驗證（claims 含 userID + email）
│   ├── handlers/
│   │   ├── auth.go               # Google OAuth callback → JWT
│   │   ├── user.go               # GET /api/me
│   │   ├── scraper.go            # TriggerScrape、PublicScrape、PushRecords、CheckOrders、UpdateTitles
│   │   ├── admin.go              # GetTitleIssues、FixSingleTitle（管理者限定）
│   │   ├── scrape_token.go       # GET /api/scrape-token
│   │   └── stats.go              # 各種統計 endpoint
│   ├── scraper/scraper.go        # 伺服器端爬蟲邏輯（goquery，舊版）
│   └── router/router.go          # 路由 + CORS
├── frontend/
│   ├── src/
│   │   ├── api/index.js          # axios + interceptors
│   │   ├── stores/auth.js        # Pinia auth store（token 存 localStorage）
│   │   ├── router/index.js       # Vue Router（未登入保護 + 管理者路由保護）
│   │   └── views/
│   │       ├── LoginView.vue
│   │       ├── AuthCallbackView.vue
│   │       ├── DashboardView.vue  # 含 MEMBERS 靜態對照表、成員排序/過濾
│   │       ├── MemberView.vue
│   │       ├── RecordsView.vue
│   │       ├── ScrapeView.vue     # 顯示 ScrapeToken
│   │       └── AdminView.vue      # タイトル未定 修正（管理者限定）
│   └── vite.config.js            # proxy: /auth/google → localhost:8080
└── extension/                    # Chrome 擴充功能
    ├── manifest.json             # Manifest V3
    ├── popup.html
    ├── popup.js
    └── popup.css
```

## 路由一覽

### Backend

| 方法 | 路徑 | 驗證 | 說明 |
|---|---|---|---|
| GET | `/auth/google` | 無 | 導向 Google OAuth |
| GET | `/auth/google/callback` | 無 | Google 回調，產生 JWT |
| POST | `/scrape` | ScrapeToken | 舊版，伺服器端爬蟲 |
| POST | `/scrape/push` | ScrapeToken | 接受擴充功能推送的已解析記錄 |
| POST | `/scrape/check-orders` | ScrapeToken | 回傳 order ID 清單中哪些是新訂單 |
| POST | `/scrape/update-titles` | ScrapeToken | 批次更新既有訂單的 single_name |
| GET | `/api/me` | JWT | 取得登入使用者資訊 |
| GET | `/api/scrape-token` | JWT | 取得/生成 ScrapeToken |
| POST | `/api/scrape` | JWT | 手動貼 Cookie 觸發爬蟲 |
| GET | `/api/records` | JWT | 取得記錄（page_size 上限 100） |
| GET | `/api/stats/overall` | JWT | 整體統計 |
| GET | `/api/stats/by-member` | JWT | 依成員統計 |
| GET | `/api/stats/by-date` | JWT | 依日期統計 |
| GET | `/api/stats/by-session` | JWT | 依場次統計 |
| GET | `/api/stats/detail` | JWT | 依 (member, single, date, session) 分組統計 |
| GET | `/api/admin/title-issues` | JWT + 管理者 | 列出 タイトル未定 問題與建議標題 |
| PUT | `/api/admin/title` | JWT + 管理者 | 修正指定 single_number 的所有 タイトル未定 紀錄 |

### Frontend

| 路徑 | 說明 | 路由保護 |
|---|---|---|
| `/` | 登入頁 | 公開 |
| `/auth/callback` | Google OAuth 回調頁 | 公開 |
| `/dashboard` | 主畫面（統計總覽） | 需登入 |
| `/member/:name` | 個別成員統計 | 需登入 |
| `/records` | 記錄列表 | 需登入 |
| `/scrape` | 爬蟲頁 | 需登入 |
| `/admin` | タイトル未定 修正 | 需登入 + 管理者 email |

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
- 擴充功能用此 token 呼叫公開的 `/scrape/*` 端點，不需要 JWT

## Chrome 擴充功能

安裝方式：`chrome://extensions/` → 開發人員模式 → 載入未封裝項目 → 選 `extension/` 資料夾

首次設定：
1. 輸入後端網址（預設 `http://localhost:8080`）
2. 從 ScrapeView 複製 ScrapeToken 貼上 → 儲存

使用方式（兩步驟）：
1. 點「同步」→ 自動開啟 `fortunemusic.jp/mypage/apply_list/` 分頁
2. 確認已登入且看到申請記錄後，點「開始抓取」

## 爬蟲邏輯（擴充功能三階段）

目標網站：`https://fortunemusic.jp`（在使用者已登入的分頁內執行，無 403 問題）

```
每一頁 apply_list 執行：

階段一 scrapeListPage（注入分頁）
  - 掃描所有 <a href> 找 /mypage/apply_detail/{id}/
  - 從 span.hdg[応募日時] 讀取應募年月
  - 從 td.tdEvent 解析：單曲號、歌名（『』）、應募次數（第N次）
  - 回傳 { orders: [{id, info}], hasMore }

階段二 check-orders（POST /scrape/check-orders）
  - 送出所有 order ID，後端比對 source_url 回傳新舊分類

階段三a fetchOrderDetails（注入分頁，僅新訂單）
  - same-origin fetch /mypage/apply_detail/{id}/
  - 解析 tbody tr：成員名【M/D 第N部】活動名 + 応募数/当選数
  - 同訂單內相同 (member, date, session) 累加後上傳 POST /scrape/push

階段三b update-titles（POST /scrape/update-titles，既有訂單）
  - 從 scrapeListPage 取得的 info 組出正確 single_name
  - 批次更新 DB 中 single_name 有變動的記錄
```

### 資料格式

**single_name**：`"41stシングル「最後に階段を駆け上がったのはいつだ？」"`（不含次數）

**lottery_round**：`"第3次"`（單獨欄位）

**event_date**：`"YYYY/M/D"`（年份由應募日期推算：活動月 < 應募月 → 隔年）

**source_url**：`"https://fortunemusic.jp/mypage/apply_detail/{id}/#member|M/D|第N部"`（每筆記錄唯一，供跨次抓取去重）

### タイトル未定 問題說明

fortunemusic 的 list page 記錄的是**應募當時**的活動名稱，不會回溯更新。  
若應募時 title 尚未公布（`『タイトル未定』`），該筆訂單的 list page 資料永遠是 タイトル未定。

- `update-titles` 無法自動修正（因為 list page 來源也是 タイトル未定）
- 解決方式：透過 `/admin` 頁面手動修正，或直接 SQL UPDATE

## 管理者機制

`handlers/admin.go` 中以 JWT claim 的 email 做判斷（`checkAdmin`），不需額外 DB 查詢。  
管理者 email 硬寫在 `adminEmail` 常數。

**`/admin` 頁面三層保護：**
1. NavBar `v-if="isAdmin"` — 連結不顯示
2. Router guard `meta: { admin: true }` — 直接打網址導回 Dashboard
3. 後端 `checkAdmin()` — API 永遠是最終防線，回 403

## DashboardView — 成員相關

`MEMBERS` 靜態對照表（`DashboardView.vue` 內）：約 87 名成員，每筆含 `{ gen, active }`。

- **排序**：依期別（gen）→ 五十音（`localeCompare('ja')`）
- **在籍過濾**：`showActiveOnly` toggle，`active: false` 為已畢業
- **不在 map 中的成員**：`gen: 99`（排最後）、`active: true`（不被過濾）
- **單曲排序**：以 `minEventDate`（最早握手會日期）為 key，讓專輯與單曲按時間軸交錯排列
- **專輯 key**：使用 `"album::<single_name>"`，避免與 `single_number: 0` 的 key 衝突

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
5. 同一訂單多商品只存第一筆 → 改用 per-order aggregated map 累加
6. 部分中選率顯示 100% → 同上，每張票各自一行需累加後再計算
7. event_date 缺少年份 → 改為 YYYY/M/D，年份由應募日期推算
8. 專輯 key 衝突 → 全部 single_number=0 的專輯合併到同一 key，改用 `"album::<name>"`
9. RecordsView 單曲/次數下拉為空 → 原本讀已移除的 `event_name` 欄位，改為讀 `single_name` / `lottery_round`
10. update-titles 對 タイトル未定 無效 → list page 歷史資料不更新，屬已知限制，透過 admin 介面手動修正
