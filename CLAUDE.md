# Fortune Tracker — 專案說明

乃木坂46 抽選統計工具。使用者以 Google 帳號登入，透過瀏覽器擴充功能自動抓取 Fortune Music 的抽選記錄並統計。

## Tech Stack

| 層 | 技術 |
|---|---|
| 前端 | Vue 3 + Vite + Element Plus + Pinia |
| 後端 | Go + Gin |
| 資料庫 | PostgreSQL（Supabase） |
| 驗證 | Google OAuth 2.0 + JWT（15 分鐘）+ Refresh Token（30 天） |

## 專案結構

```
fortunemusic/
├── backend/
│   ├── main.go
│   ├── config/config.go          # 讀取 .env（含 ADMIN_EMAIL）
│   ├── db/db.go                  # Gorm + PostgreSQL + AutoMigrate + lottery_round migration
│   ├── models/
│   │   ├── user.go               # User{ID, GoogleID, Email, Name, ScrapeToken, RefreshToken, RefreshTokenExpiry}
│   │   ├── record.go             # 個握抽選記錄（含 order_id index，lottery_round int）
│   │   ├── full_record.go        # 全握記錄（FullRecord，獨立資料表）
│   │   ├── purchase.go           # 購入花費記錄（lottery_round int，applied_at）
│   │   └── scrape_log.go         # 抓取記錄（type, new_count, skip_count, error）
│   ├── middleware/auth.go        # JWT 驗證（claims 含 userID + email）
│   ├── handlers/
│   │   ├── auth.go               # Google OAuth → opaque code → JWT + refresh token
│   │   ├── user.go               # GET /api/me（含 is_admin）
│   │   ├── scraper.go            # TriggerScrape、PublicScrape、PushRecords、CheckOrders、UpdateTitles
│   │   ├── full_scraper.go       # PushFullRecords、GetFullRecords
│   │   ├── full_stats.go         # GetFullOverallStats、GetFullStatsByMember、GetFullStatsBySingle
│   │   ├── admin.go              # GetTitleIssues、FixSingleTitle、GetAdminUsers、DeleteUserRecords、DeleteUserFullRecords
│   │   ├── purchase_scraper.go   # CheckEntries、PushPurchases、GetPurchases、GetPurchaseTree、購入統計
│   │   ├── scrape_log.go         # PushScrapeLog、GetAdminScrapeLogs
│   │   ├── scrape_token.go       # GET /api/scrape-token
│   │   └── stats.go              # 個握各種統計 endpoint
│   ├── scraper/scraper.go        # 伺服器端爬蟲邏輯（goquery，舊版）
│   └── router/router.go          # 路由 + CORS（含正式環境 URL）
├── frontend/
│   ├── src/
│   │   ├── api/index.js          # axios + interceptors（401 自動 refresh）
│   │   ├── stores/auth.js        # Pinia auth store（token + refreshToken 存 localStorage）
│   │   ├── router/index.js       # Vue Router（未登入保護 + 管理者路由保護）
│   │   ├── utils/members.js      # MEMBERS 常數（87名、gen+active）+ sortMembersByGen
│   │   └── views/
│   │       ├── LoginView.vue
│   │       ├── AuthCallbackView.vue  # 用 code 換 JWT + refresh token
│   │       ├── DashboardView.vue
│   │       ├── MemberView.vue
│   │       ├── RecordsView.vue       # 個握紀錄列表
│   │       ├── FullView.vue          # 全握統計頁（總覽、依類型/場地、依成員、依單曲）
│   │       ├── ScrapeView.vue        # 顯示 ScrapeToken
│   │       ├── SpendingView.vue      # 購入花費統計（單曲→抽次→成員樹狀）
│   │       └── AdminView.vue         # タイトル未定 修正 + 使用者管理 + 刪除資料 + 抓取紀錄（管理者限定）
│   └── vite.config.js            # proxy: /auth/google → localhost:8080
└── extension/                    # Chrome 擴充功能
    ├── manifest.json             # Manifest V3（含 ticket.fortunemeets.app host_permissions）
    ├── popup.html                # 個握 + 全握操作區塊
    ├── popup.js
    └── popup.css
```

## 路由一覽

### Backend

| 方法 | 路徑 | 驗證 | 說明 |
|---|---|---|---|
| GET | `/auth/google` | 無 | 導向 Google OAuth |
| GET | `/auth/google/callback` | 無 | Google 回調，產生 opaque code |
| POST | `/auth/token` | 無 | 用 opaque code 換 JWT + refresh token |
| POST | `/auth/refresh` | 無 | 用 refresh token 換新 JWT |
| POST | `/scrape` | ScrapeToken | 舊版，伺服器端爬蟲 |
| POST | `/scrape/push` | ScrapeToken | 接受擴充功能推送的已解析記錄 |
| POST | `/scrape/check-orders` | ScrapeToken | 回傳 order ID 清單中哪些是新訂單 |
| POST | `/scrape/update-titles` | ScrapeToken | 批次更新既有訂單的 single_name |
| GET | `/api/me` | JWT | 取得登入使用者資訊（含 is_admin） |
| GET | `/api/scrape-token` | JWT | 取得/生成 ScrapeToken |
| POST | `/api/scrape` | JWT | 手動貼 Cookie 觸發爬蟲 |
| GET | `/api/records` | JWT | 取得記錄（page_size 上限 100） |
| GET | `/api/stats/overall` | JWT | 整體統計 |
| GET | `/api/stats/by-member` | JWT | 依成員統計 |
| GET | `/api/stats/by-date` | JWT | 依日期統計 |
| GET | `/api/stats/by-session` | JWT | 依場次統計 |
| GET | `/api/stats/detail` | JWT | 依 (member, single, date, session) 分組統計 |
| GET | `/api/stats/order-sequence` | JWT | 各張應募訂單序號中選率（SQL window function） |
| GET | `/api/admin/title-issues` | JWT + 管理者 | 列出 タイトル未定 問題與建議標題 |
| PUT | `/api/admin/title` | JWT + 管理者 | 修正指定 single_number 的所有 タイトル未定 紀錄 |
| GET | `/api/admin/users` | JWT + 管理者 | 列出所有使用者 |
| DELETE | `/api/admin/users/:id/records` | JWT + 管理者 | 刪除指定使用者個握記錄（支援 single_number / date_from / date_to 篩選） |
| DELETE | `/api/admin/users/:id/full-records` | JWT + 管理者 | 刪除指定使用者全握記錄（支援 single_number / date_from / date_to 篩選） |
| DELETE | `/api/admin/users/:id/purchases` | JWT + 管理者 | 刪除指定使用者購入記錄 |
| GET | `/api/admin/scrape-logs` | JWT + 管理者 | 最近 50 筆抓取記錄（含使用者、類型、新增/跳過數、錯誤訊息） |
| POST | `/scrape/full/push` | ScrapeToken | 接受擴充功能推送的全握記錄（dedup by order_id） |
| POST | `/scrape/check-entries` | ScrapeToken | 回傳 entry ID 清單中哪些是新的（購入用） |
| POST | `/scrape/purchases/push` | ScrapeToken | 接受擴充功能推送的購入記錄 |
| POST | `/scrape/log` | ScrapeToken | 記錄一次抓取操作結果（type/new_count/skip_count/error） |
| GET | `/api/purchases` | JWT | 購入記錄列表（分頁） |
| GET | `/api/purchases/tree` | JWT | 購入樹狀統計（單曲→抽次→成員，依 min applied_at DESC） |
| GET | `/api/purchases/stats/overall` | JWT | 購入整體統計 |
| GET | `/api/purchases/stats/by-single` | JWT | 購入依單曲統計 |
| GET | `/api/purchases/stats/by-member` | JWT | 購入依成員統計 |
| GET | `/api/full/records` | JWT | 取得全握記錄（分頁，支援 member/event_type/venue/single_number 篩選） |
| GET | `/api/full/stats/overall` | JWT | 全握整體統計 + 依類型/場地分組 |
| GET | `/api/full/stats/by-member` | JWT | 全握依成員統計 |
| GET | `/api/full/stats/by-single` | JWT | 全握依單曲統計 |

### Frontend

| 路徑 | 說明 | 路由保護 |
|---|---|---|
| `/` | 登入頁 | 公開 |
| `/auth/callback` | Google OAuth 回調頁 | 公開 |
| `/dashboard` | 主畫面（統計總覽） | 需登入 |
| `/member/:name` | 個別成員統計 | 需登入 |
| `/records` | 個握紀錄列表 | 需登入 |
| `/full` | 全握統計頁 | 需登入 |
| `/scrape` | 爬蟲頁 | 需登入 |
| `/spending` | 購入花費統計 | 需登入 |
| `/admin` | タイトル未定 修正 + 使用者管理 + 刪除資料 + 抓取紀錄 | 需登入 + 管理者 |

## 驗證流程

```
LoginView → GET http://localhost:8080/auth/google
  → Google OAuth
  → GET /auth/google/callback（後端）
  → 產生 opaque code（60秒效期，存 sync.Map）
  → redirect 前端 /auth/callback?code=...
  → AuthCallbackView POST /auth/token 換 JWT + refresh token
  → 存入 localStorage → redirect 到原始目標路由
```

**OAuth state 保護：** 每次登入產生隨機 nonce，以 HMAC-SHA256 簽名，callback 時驗證，防 CSRF。

**注意：** Vite proxy 只代理 `/auth/google`，不代理 `/auth/callback`（否則會 404）。

## Token 機制

### JWT（Access Token）
- 有效期 15 分鐘，存 localStorage
- 過期時 axios 401 interceptor 自動呼叫 `/auth/refresh` 換新 JWT，使用者無感知
- Refresh 失敗才清除 token 並導回登入頁

### Refresh Token
- 有效期 30 天，存 localStorage + DB（`users.refresh_token`）
- 登入時由 `/auth/token` 一起回傳
- 30 天內不需重新登入

### ScrapeToken
- 長期有效，存 DB，由使用者手動複製到擴充功能
- 用於 `/scrape/*` 公開端點，不需 JWT

## Chrome 擴充功能

安裝方式：`chrome://extensions/` → 開發人員模式 → 載入未封裝項目 → 選 `extension/` 資料夾

首次設定：
1. 輸入後端網址（預設 `http://localhost:8080`）
2. 從 ScrapeView 複製 ScrapeToken 貼上 → 儲存

使用方式 — 個握（兩步驟）：
1. 點「同步」→ 自動開啟 `fortunemusic.jp/mypage/apply_list/` 分頁
2. 確認已登入且看到申請記錄後，點「開始抓取」

使用方式 — 全握（兩步驟）：
1. 輸入起訖單曲號（如 41～41），點「全握同步」→ 開啟 `ticket.fortunemeets.app/nogizaka46/Nst#/history`
2. 確認已登入且看到歷史記錄後，點「開始全握抓取」（自動依序掃描各單，連續 3 個空頁停止）

## 爬蟲邏輯（擴充功能）

### 個握三階段

目標網站：`https://fortunemusic.jp`（在使用者已登入的分頁內執行，無 403 問題）

```
每一頁 apply_list 執行：

階段一 scrapeListPage（注入分頁）
  - 掃描所有 <a href> 找 /mypage/apply_detail/{id}/
  - 從 span.hdg[応募日時] 讀取應募年月
  - 從 td.tdEvent 解析：單曲號、歌名（『』）、應募次數（第N次）
  - 回傳 { orders: [{id, info}], hasMore }

階段二 check-orders（POST /scrape/check-orders）
  - 送出所有 order ID，後端比對 order_id 欄位回傳新舊分類（批次查詢）

階段三a fetchOrderDetails（注入分頁，僅新訂單，4 個並行）
  - same-origin fetch /mypage/apply_detail/{id}/
  - 解析 tbody tr：成員名【M/D 第N部】活動名 + 応募数/当選数
  - 同訂單內相同 (member, date, session) 累加後上傳 POST /scrape/push

階段三b update-titles（POST /scrape/update-titles，既有訂單）
  - 從 scrapeListPage 取得的 info 組出正確 single_name
  - 批次更新 DB 中 single_name 有變動的記錄
```

### 資料格式

**single_name**：`"41stシングル「最後に階段を駆け上がったのはいつだ？」"`（不含次數）

**lottery_round**：整數 `3`（DB 欄位型別為 integer；舊版字串 "第3次" 已透過 migration 轉換）

**event_date**：`"YYYY/M/D"`（年份由應募日期推算：活動月 < 應募月 → 隔年）

**source_url**：`"https://fortunemusic.jp/mypage/apply_detail/{id}/#member|M/D|第N部"`（每筆記錄唯一，供去重）

**order_id**：從 source_url 提取的訂單 ID，有 DB index，供 CheckOrders 批次查詢

### 全握爬蟲（scrapeFullPage）

目標網站：`https://ticket.fortunemeets.app`（SPA，Vue Router hash 模式）

URL 格式：`/nogizaka46/{N}{suffix}#/history`（suffix = st/nd/rd/th，正確處理 11th/12th/13th）

```
DOM 結構（實測）：
  div.result.win / div.result.lose        ← 主行容器
    div.result-body
      span.tag.win / span.tag.lose        ← 当選 / 落選 標籤
      div > div
        p    2023年11月19日（日）＠場地   ← 日期行（全形字元需 toHalf 轉換）
        p    第１部                        ← 場次（全形數字）
        p    成員名                        ← 成員
    div.flex-shrink-0   5枚（5口）         ← 應募口數

落選行外層多包一層 div.resultWrap，但 div.result.lose 仍可直接 querySelectorAll('div.result') 取得。
```

**toHalf()** 轉換全形字元（U+FF01–U+FF5E → U+0021–U+007E），用於日期/場次解析前處理。

**dedup key**：`${singleNum}:${eventType}:${venue}:${eventDate}:${session}:${memberName}:${status}`
- 包含 status（当選/落選）確保同成員同場次的不同結果不被合併
- 同一場次有當選也有落選時（一抽/二抽）目前無法自動區分，為已知限制

**event_type**：日期行含 `@` → `実体`；否則 → `線上`

**venue**：`@` 後含「幕張/東京/Makuhari」→ `東京`；否則 → `地方`

### タイトル未定 問題說明

fortunemusic 的 list page 記錄的是**應募當時**的活動名稱，不會回溯更新。  
若應募時 title 尚未公布（`『タイトル未定』`），該筆訂單的 list page 資料永遠是 タイトル未定。

- `update-titles` 無法自動修正（因為 list page 來源也是 タイトル未定）
- 解決方式：透過 `/admin` 頁面手動修正，或直接 SQL UPDATE

## 管理者機制

`handlers/admin.go` 中以 JWT claim 的 email 做判斷（`checkAdmin`）。  
管理者 email 從 `ADMIN_EMAIL` 環境變數讀取，不寫在程式碼中。

**`/admin` 頁面三層保護：**
1. NavBar `v-if="isAdmin"` — 連結不顯示（`is_admin` 由 `/api/me` 回傳）
2. Router guard `meta: { admin: true }` — 直接打網址導回 Dashboard
3. 後端 `checkAdmin()` — API 永遠是最終防線，回 403

**刪除資料功能（AdminView）：**
- 模式下拉：清除某人全部資料 / 清除特定單曲 / 清除特定日期範圍
- 類型下拉：個握（records）/ 全握（full-records）
- `buildDeleteQuery` helper 統一處理兩種資料表的刪除邏輯

## 成員資料（src/utils/members.js）

`MEMBERS` 靜態對照表：約 87 名成員（1期–6期），每筆含 `{ gen, active }`。

- **排序**：`sortMembersByGen(names)` 依期別（gen）→ 五十音（`localeCompare('ja')`）
- **在籍過濾**：`showActiveOnly` toggle，`active: false` 為已畢業
- **不在 map 中的成員**：`gen: 99`（排最後）、`active: true`（不被過濾）
- **所有下拉選單**（DashboardView、RecordsView）統一從此 import，不各自維護

## DashboardView — 成員手風琴

- **單曲排序**：以 `minEventDate`（最早握手會日期）為 key，讓專輯與單曲按時間軸交錯排列
- **專輯 key**：使用 `"album::<single_name>"`，避免與 `single_number: 0` 的 key 衝突

## SpendingView — 花費頁面

- **樹狀結構**：單曲/專輯 → 抽次 → 成員，三層 el-collapse 展開
- **排序**：以每個單曲/專輯的最早 `applied_at`（購入申請時間）DESC 排列，新的在前，專輯與單曲自然依時間交錯
- **抽次排序**：`lottery_round`（整數）ASC，確保 1抽、2抽、12抽 等正確數字順序
- **formatRound**：直接用整數值顯示，如 `3` → `"3抽"`

## .env 設定（backend）

```
DATABASE_URL=...
GOOGLE_CLIENT_ID=...
GOOGLE_CLIENT_SECRET=...
JWT_SECRET=...
APP_URL=http://localhost:8080
FRONTEND_URL=http://localhost:5173
ADMIN_EMAIL=...
```

`FRONTEND_URL` 同時用於 CORS 白名單，部署正式環境時改此值即可。

## 已修正的問題

1. Vite proxy `/auth` 太寬，改成 `/auth/google`（否則 `/auth/callback` 會被導到後端 → 404）
2. JWT 過期無聲失敗 → 加 401 interceptor，自動 refresh；refresh 失敗才提示重登入
3. `page_size: 500` 超過後端上限 100 被靜默忽略 → 改成 100
4. 登入後未保留原始目標路由 → 用 localStorage 暫存，登入後跳回
5. 同一訂單多商品只存第一筆 → 改用 per-order aggregated map 累加
6. 部分中選率顯示 100% → 同上，每張票各自一行需累加後再計算
7. event_date 缺少年份 → 改為 YYYY/M/D，年份由應募日期推算
8. 專輯 key 衝突 → 全部 single_number=0 的專輯合併到同一 key，改用 `"album::<name>"`
9. RecordsView 單曲/次數下拉為空 → 原本讀已移除的 `event_name` 欄位，改為讀 `single_name` / `lottery_round`
10. update-titles 對 タイトル未定 無效 → list page 歷史資料不更新，屬已知限制，透過 admin 介面手動修正
11. OAuth state 固定值無 CSRF 保護 → 改為 HMAC-SHA256 簽名的隨機 nonce
12. JWT 透過 URL 傳遞 → 改用 opaque code exchange，JWT 不出現在 URL
13. 無 refresh token → 加入 30 天 refresh token，15 分鐘 JWT 到期自動續期
14. Admin email 硬寫在程式碼 → 改從 ADMIN_EMAIL 環境變數讀取
15. CheckOrders N+1 查詢 → 改成單次批次查詢
16. source_url LIKE 全表掃描 → 加 order_id 欄位 + index，改用 = 查詢
17. GetOrderSequenceStats 全量撈入記憶體 → 改用 SQL window function（CTE）
18. CORS 只有 localhost → 加入 cfg.FrontendURL，部署正式環境改 .env 即可
19. Extension 串行抓取 detail page → 改為 4 個並行一批，速度提升約 4 倍
20. 全握功能上線 → 新增 FullRecord 資料表、全握 API、FullView 統計頁、Extension 全握爬蟲
21. Admin 刪除功能細化 → 從「刪除某人全部資料」改為支援按單曲號/日期範圍精確刪除，同時支援個握/全握
22. 全握爬蟲 dedup key 加入 status → 避免同成員同場次的当選/落選記錄被錯誤合併
23. lottery_round 改為整數型別 → 修正字串排序造成 "第12次" 排在 "第2次" 前的 bug；DB migration 冪等轉換；Extension、前端、後端全面同步
24. SpendingView 單曲排序改為 min(applied_at) DESC → 專輯（single_number=0）與單曲依最早購買時間交錯排列，新的在前
25. 新增後端抓取紀錄（ScrapeLog） → 新增 scrape_logs 資料表、POST /scrape/log 端點、GET /api/admin/scrape-logs；共用帳號出問題時管理者可從 AdminView 查閱各使用者的抓取結果、新增/跳過數與錯誤訊息
26. Extension 抓取進度條與操作記錄 → 取代舊版單行 showResult；抓取中顯示進度列（個握/購入為不確定式、全握有終止頁時為確定式）；每次操作結束後在 popup 內追加一筆記錄（成功/錯誤/警告），關閉 popup 後清除
