# Fortune Tracker — 專案說明

坂道系（乃木坂46 / 櫻坂46 / 日向坂46）抽選統計工具。使用者以 Google 帳號登入，透過瀏覽器擴充功能自動抓取 Fortune Music 的抽選記錄並統計。

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
│   │   ├── user.go               # User{ID, GoogleID, Email, Name, ScrapeToken, RefreshToken, RefreshTokenExpiry}（無 is_admin 欄位，由 ADMIN_EMAIL 比對運算）
│   │   ├── record.go             # 個握抽選記錄（含 order_id index、lottery_round int、group）
│   │   ├── full_record.go        # 全握記錄（FullRecord，獨立資料表，含 group）
│   │   ├── sign_event.go         # 簽名會記錄（SignEvent，order_id 含 "_sign" 後綴區分，含 group）
│   │   ├── purchase.go           # 購入花費記錄（lottery_round int，applied_at，item_key 唯一索引，group）
│   │   ├── scrape_log.go         # 抓取記錄（type, new_count, skip_count, error, duration_sec）
│   │   ├── title.go              # 單曲名稱主表（Title model，對應 DB 表 titles，(group, single_number) 複合唯一鍵 → single_name，供自動套用）
│   │   └── venue.go              # 場地對照表（Venue model，對應 DB 表 venues，(group, single_number, event_date) 複合唯一鍵 → venue_name，供自動套用）
│   ├── middleware/auth.go        # JWT 驗證（claims 含 userID + email）
│   ├── handlers/
│   │   ├── auth.go               # Google OAuth → opaque code → JWT + refresh token
│   │   ├── user.go               # GET /api/me（含 is_admin）
│   │   ├── scraper.go            # TriggerScrape、PublicScrape、PushRecords（批次查重+批次寫入）、CheckOrders、UpdateTitles
│   │   ├── full_scraper.go       # PushFullRecords（全握+簽名會共用，批次查重+批次寫入）、GetFullRecords、GetSignEvents
│   │   ├── full_stats.go         # GetFullOverallStats、GetFullStatsByMember、GetFullStatsBySingle、GetFullDetailStats（venue 列入 GROUP BY）
│   │   ├── admin.go              # GetTitleIssues、FixSingleTitle、BulkSetTitles、GetAdminUsers、Preview/Delete User{Records,FullRecords,Purchases,SignEvents}、GetAdminSignEvents
│   │   ├── purchase_scraper.go   # CheckEntries、PushPurchases（批次查重+批次寫入）、GetPurchases、GetPurchaseTree、購入統計
│   │   ├── scrape_log.go         # PushScrapeLog、GetAdminScrapeLogs
│   │   ├── scrape_token.go       # GET /api/scrape-token
│   │   └── stats.go              # 個握各種統計 endpoint（含 GetDetailStats、GetOrderSequenceStats）
│   ├── middleware/auth.go        # JWT 驗證（claims 含 userID + email）
│   ├── middleware/impersonate.go # 管理者代理：X-Impersonate-User header 可切換操作身份
│   ├── scraper/scraper.go        # 伺服器端爬蟲邏輯（goquery，舊版）
│   └── router/router.go          # 路由 + CORS（含正式環境 URL + chrome-extension://）
├── frontend/
│   ├── src/
│   │   ├── api/index.js          # axios + interceptors（401 自動 refresh）
│   │   ├── stores/
│   │   │   ├── auth.js           # Pinia auth store（token + refreshToken 存 localStorage）
│   │   │   ├── data.js           # hasData 狀態，check() 呼叫 GetStats 判斷使用者是否已有資料（供路由判斷導向設定頁）
│   │   │   ├── impersonate.js    # 管理者「模擬畫面」狀態（start/stop），搭配 X-Impersonate-User header
│   │   │   └── theme.js          # 深色模式（isDark）+ currentMember 主題色
│   │   ├── router/index.js       # Vue Router（未登入保護 + 管理者路由保護 + DATA_ROUTES 資料檢查）
│   │   ├── utils/members.js      # MEMBERS 常數（87名、gen+active）+ sortMembersByGen
│   │   └── views/
│   │       ├── LoginView.vue
│   │       ├── AuthCallbackView.vue  # 用 code 換 JWT + refresh token，並靜默觸發擴充功能連結
│   │       ├── SetupView.vue         # 擴充功能未安裝時的三步驟安裝引導
│   │       ├── DashboardView.vue
│   │       ├── MemberView.vue
│   │       ├── RecordsView.vue       # 個握紀錄列表
│   │       ├── RecordsAnalysisView.vue # 個握分析（建設中，僅佔位）
│   │       ├── SpendingView.vue      # 個握花費統計（單曲→抽次→成員樹狀）
│   │       ├── FullView.vue          # 全握紀錄列表（含 group/member/type/venue/single/round 篩選）
│   │       ├── FullAnalysisView.vue  # 全握分析（總覽卡 + 類型分析 + 成員統計 + 成員詳細分析三個摺疊面板）
│   │       ├── FullSpendingView.vue  # 全握花費（建設中，僅佔位）
│   │       ├── ScrapeView.vue        # 同步工具設定（一鍵連結擴充功能至帳號，失敗時顯示安裝步驟）
│   │       ├── MaintenanceView.vue   # 維護模式畫面（App.vue 偵測 /status.json 後條件渲染）
│   │       ├── AdminUsersView.vue       # 使用者管理（模擬畫面）（管理者限定）
│   │       ├── AdminMaintenanceView.vue # 刪除資料(含預覽) + 抓取紀錄（管理者限定）
│   │       ├── AdminTitlesView.vue      # 單曲名稱：タイトル未定修正 + 批次登記 + 已知單曲名稱瀏覽（管理者限定）
│   │       ├── AdminVenuesView.vue      # 場地管理：缺少場地修正 + 批次登記 + 已知場地瀏覽（管理者限定）
│   │       └── AdminSignEventsView.vue  # 簽名會紀錄（管理者限定）
│   └── vite.config.js            # proxy: /auth/google → localhost:8080
└── extension/                    # Chrome 擴充功能
    ├── manifest.json             # Manifest V3（含 ticket.fortunemeets.app host_permissions）
    ├── background.js             # Service worker，監聽 onMessageExternal 的 FORTUNE_SETUP 訊息（一鍵連結帳號）
    ├── popup.html                # 個握 + 全握操作區塊（全握含 #fullGroup 分組下拉）
    ├── popup.js                  # 個握 group 由事件文字自動判斷；全握 group 由 #fullGroup 下拉手動選擇
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
| GET | `/api/stats` | JWT | 整體統計（同 overall，舊版相容端點） |
| GET | `/api/stats/overall` | JWT | 整體統計 |
| GET | `/api/stats/by-member` | JWT | 依成員統計 |
| GET | `/api/stats/by-date` | JWT | 依日期統計 |
| GET | `/api/stats/by-session` | JWT | 依場次統計 |
| GET | `/api/stats/detail` | JWT | 依 (member, single, date, session) 分組統計 |
| GET | `/api/stats/order-sequence` | JWT | 各張應募訂單序號中選率（SQL window function） |
| GET | `/api/admin/title-issues` | JWT + 管理者 | 列出 タイトル未定 問題與建議標題（依 group + single_number 分組，不同團體同號不互相混淆） |
| PUT | `/api/admin/title` | JWT + 管理者 | 修正指定 (group, single_number) 的所有 タイトル未定 紀錄 |
| POST | `/api/admin/titles/bulk` | JWT + 管理者 | 批次登記多筆已知單曲名稱（不需先出現問題），同時回填既有 タイトル未定 紀錄 |
| GET | `/api/admin/titles` | JWT + 管理者 | 列出 DB 裡所有已知單曲名稱（依 group 分組，合併 titles 主表 + records/purchases 推測，標示來源） |
| GET | `/api/admin/venue-issues` | JWT + 管理者 | 列出全握実体場次裡 venue 空白的 (group, single_number, event_date) 組合與建議場地 |
| PUT | `/api/admin/venue` | JWT + 管理者 | 登記指定 (group, single_number, event_date) 的場地，回填既有空白紀錄並寫入 venues 表供未來自動套用 |
| POST | `/api/admin/venues/bulk` | JWT + 管理者 | 批次登記多筆已知場地（不需先出現問題），同時回填既有空白紀錄 |
| GET | `/api/admin/venues` | JWT + 管理者 | 列出 DB 裡所有已知場地（合併 venues 表登記 + full_records 已有場地文字的實際資料，標示來源） |
| GET | `/api/admin/users` | JWT + 管理者 | 列出所有使用者 |
| GET | `/api/admin/users/:id/records/preview` | JWT + 管理者 | 刪除前預覽指定使用者個握記錄（分頁，支援 group/single_number/date_from/date_to 篩選） |
| GET | `/api/admin/users/:id/full-records/preview` | JWT + 管理者 | 刪除前預覽指定使用者全握記錄（同上篩選） |
| GET | `/api/admin/users/:id/purchases/preview` | JWT + 管理者 | 刪除前預覽指定使用者購入記錄（分頁，支援 single_number/date_from/date_to 篩選） |
| DELETE | `/api/admin/users/:id/records` | JWT + 管理者 | 刪除指定使用者個握記錄（支援 group / single_number / date_from / date_to 篩選） |
| DELETE | `/api/admin/users/:id/full-records` | JWT + 管理者 | 刪除指定使用者全握記錄（支援 group / single_number / date_from / date_to 篩選） |
| DELETE | `/api/admin/users/:id/purchases` | JWT + 管理者 | 刪除指定使用者購入記錄 |
| GET | `/api/admin/scrape-logs` | JWT + 管理者 | 最近 50 筆抓取記錄（含使用者、類型、新增/跳過數、錯誤訊息、耗時） |
| GET | `/api/admin/sign-events` | JWT + 管理者 | 所有使用者的簽名會紀錄（分頁，支援 user_id/member/single_number 篩選） |
| GET | `/api/admin/users/:id/sign-events/preview` | JWT + 管理者 | 刪除前預覽指定使用者簽名會記錄（分頁，支援 group/single_number/date_from/date_to 篩選） |
| DELETE | `/api/admin/users/:id/sign-events` | JWT + 管理者 | 刪除指定使用者簽名會記錄（支援 group/single_number/date_from/date_to 篩選） |
| GET | `/api/sign-events` | JWT | 取得個人簽名會記錄（分頁，支援 group/member/single_number 篩選） |
| POST | `/scrape/full/push` | ScrapeToken | 接受擴充功能推送的全握記錄與簽名會記錄（依 order_id 是否含 `_sign` 分流，批次查重） |
| POST | `/scrape/check-entries` | ScrapeToken | 回傳 entry ID 清單中哪些是新的（購入用） |
| POST | `/scrape/purchases/push` | ScrapeToken | 接受擴充功能推送的購入記錄（批次查重） |
| POST | `/scrape/log` | ScrapeToken | 記錄一次抓取操作結果（type/new_count/skip_count/error/duration_sec） |
| GET | `/api/purchases` | JWT | 購入記錄列表（分頁） |
| GET | `/api/purchases/tree` | JWT | 購入樹狀統計（團體→單曲→抽次→成員，各層都依 min applied_at DESC） |
| GET | `/api/purchases/stats/overall` | JWT | 購入整體統計 |
| GET | `/api/purchases/stats/by-single` | JWT | 購入依單曲統計 |
| GET | `/api/purchases/stats/by-member` | JWT | 購入依成員統計 |
| GET | `/api/full/records` | JWT | 取得全握記錄（分頁，支援 group/member/event_type/venue/single_number/lottery_round 篩選） |
| GET | `/api/full/stats/overall` | JWT | 全握整體統計 + 依類型/場地分組 |
| GET | `/api/full/stats/by-member` | JWT | 全握依成員統計 |
| GET | `/api/full/stats/by-single` | JWT | 全握依單曲統計 |
| GET | `/api/full/stats/detail` | JWT | 全握依 (single, venue, session, member, lottery_round) 詳細分析，供成員詳細分析表格使用（venue 列入 GROUP BY，同張單不同場地分開計） |

### Frontend

| 路徑 | 說明 | 路由保護 |
|---|---|---|
| `/` | 登入頁 | 公開 |
| `/auth/callback` | Google OAuth 回調頁 | 公開 |
| `/setup` | 擴充功能安裝引導（三步驟） | 需登入 |
| `/dashboard` | 主畫面（統計總覽） | 需登入，會觸發 data store check() |
| `/member/:name` | 個別成員統計 | 需登入 |
| `/records` | 個握紀錄列表 | 需登入，會觸發 data store check() |
| `/records/analysis` | 個握分析（建設中） | 需登入 |
| `/spending` | 個握花費統計 | 需登入，會觸發 data store check() |
| `/full` | 全握紀錄列表 | 需登入 |
| `/full/spending` | 全握花費（建設中） | 需登入 |
| `/full/analysis` | 全握分析（總覽/類型/成員/成員詳細） | 需登入 |
| `/full/sign-events` | 個人簽名會紀錄 | 需登入 |
| `/scrape` | 爬蟲頁 | 需登入 |
| `/admin` | redirect → `/admin/users` | 需登入 + 管理者 |
| `/admin/users` | 使用者管理（模擬畫面） | 需登入 + 管理者 |
| `/admin/maintenance` | 刪除資料(預覽) + 抓取紀錄 | 需登入 + 管理者 |
| `/admin/titles` | 單曲名稱：タイトル未定修正 + 批次登記 + 已知名稱瀏覽 | 需登入 + 管理者 |
| `/admin/venues` | 場地管理：缺少場地修正 + 批次登記 + 已知場地瀏覽 | 需登入 + 管理者 |
| `/admin/sign-events` | 簽名會紀錄 | 需登入 + 管理者 |
| `/:pathMatch(.*)*` | redirect → `/dashboard` | 需登入 |

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

下載：`https://github.com/MaoIsCute/FortuneMusic/raw/main/FTExtension.zip`

**重要：每次修改 `extension/` 底下任何檔案後，commit 前都要重新打包 `FTExtension.zip`**（直接壓縮 `extension/` 資料夾內容，檔案要在 zip 根目錄、不要多包一層 `extension` 資料夾——用 `Compress-Archive -Path extension\* -DestinationPath FTExtension.zip -Force`），否則使用者下載到的會是舊版擴充功能。

**版本號：commit 前若有異動，各自更新：**
- 有改 `extension/` 檔案 → 更新 `extension/manifest.json` 的 `"version"`（目前 `1.4`）
- 有改 `frontend/` 檔案 → 更新 `frontend/src/components/NavBar.vue` 的 `APP_VERSION`（目前 `1.4`）

兩者互相獨立，不需強制同步。

安裝方式：`chrome://extensions/` → 開發人員模式 → 載入未封裝項目 → 選解壓縮後的資料夾

首次設定（一鍵連結）：
1. 以 Google 帳號登入網站
2. 登入後自動嘗試連結擴充功能（`AuthCallbackView` 靜默發送 `FORTUNE_SETUP`）
3. 若擴充功能未安裝 → 導向 `/setup` 顯示三步驟安裝引導
4. 安裝完成後點「連結帳號」按鈕即完成，不需手動複製 ScrapeToken

使用方式 — 個握（兩步驟）：
1. 點「同步」→ 自動開啟 `fortunemusic.jp/mypage/apply_list/` 分頁
2. 確認已登入且看到申請記錄後，點「開始抓取」

使用方式 — 全握（兩步驟）：
1. 選擇分組（`#fullGroup` 下拉，預設乃木坂46），輸入起訖單曲號（如 41～41），點「全握同步」→ 開啟 `ticket.fortunemeets.app/{group}/Nst#/history`
2. 確認已登入且看到歷史記錄後，點「開始全握抓取」（自動依序掃描各單，連續 3 個空頁停止）

## 爬蟲邏輯（擴充功能）

### 個握三階段

目標網站：`https://fortunemusic.jp`（在使用者已登入的分頁內執行，無 403 問題）

```
每一頁 apply_list 執行：

階段一 scrapeListPage（注入分頁）
  - 掃描所有 <a href> 找 /mypage/apply_detail/{id}/
  - 從 span.hdg[応募日時] 讀取應募年月
  - 從 td.tdEvent 解析：單曲號、歌名（『』）、應募次數（第N次）、group（依文字含「乃木坂46/櫻坂46/日向坂46」判斷）
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

**venue**：直接取 `@` 後的原始文字 `trim()`（例：「幕張メッセ」），**沒有**簡化成「東京/地方」——之前文件寫錯，目前 `popup.js` 的 `dateStr.slice(atIdx + 1).trim()` 就是最終存進 DB 的值，可能是任意長度的場地全名，UI 顯示場地的表格欄位需要給足夠寬度（或允許換行），不能假設只有 2 個字

### 購入花費爬蟲（fetchEntryDetailItems）

目標網站：`https://fortunemusic.jp/mypage/entry_detail/{id}/`（購入訂單明細頁）

- `item_key` = `entry_id:member_name:event_date:session`，沒有列項序號，所以同一張訂單內若有多個 member+date+session 完全相同的 `<tr>`（例：分兩次加購同一場次同一成員，各 3 個，頁面小計 7,200円），早期版本會各自送出一筆，後端依 item_key 去重時把第二筆當成「已存在」整筆漏算
- **修正**：`fetchSingle()` 在擷取階段先依 member_name+date+session 把同 entry 內的列項加總（quantity/subtotal 相加）成一筆再送出，邏輯與個握 `fetchOrderDetails()` 的合併方式一致；同一 entry_id 之後若被重新抓取，加總結果的 item_key 不變，仍會被既有的存在性檢查正確跳過，不會重複計算
- **小計驗證**：解析時同步讀取頁面顯示的「小計」文字，跟加總後的 subtotal 比對；不一致代表**這次抓取本身漏抓了某些 `<tr>`**，會在 popup 操作記錄追加一筆 ⚠️ 警告（含 entry_id），不會擋住正常流程
- **已知限制**：小計驗證只在抓取當下比對，無法回頭驗證這次修正**之前**就已經存進 DB 的舊資料（那些 entry_id 已存在於 DB，正常同步與「補抓遺漏」都不會再重新抓取它們）。要稽核舊資料需要額外做「強制重抓所有已存在 entry 並與 DB 比對」的功能，目前尚未實作

### タイトル未定 問題說明

fortunemusic 的 list page 記錄的是**應募當時**的活動名稱，不會回溯更新。  
若應募時 title 尚未公布（`『タイトル未定』`），該筆訂單的 list page 資料永遠是 タイトル未定。

- `update-titles` 無法自動修正（因為 list page 來源也是 タイトル未定）
- 解決方式：透過 `/admin/titles` 頁面的 `PUT /api/admin/title` 手動修正，或 `POST /api/admin/titles/bulk` 批次登記
- `FixSingleTitle` 會同時更新所有使用者的個握（records）與購入（purchases）資料
- 修正結果同時寫入 `titles` 資料表（model: `Title`），供後續新抓取自動套用建議名稱

**`titles` 表的角色定位**：原本叫 `title_corrections`（被動的「タイトル未定修正對照表」），後來改名為 `titles`（model `models.Title`），改成主動維護的「單曲名稱主表」——使用者直接透過批次登記把已知的單曲名稱整批寫進去，不用等系統偵測到 タイトル未定 問題才被動修正。`/admin/titles` 的「已知單曲名稱」面板（`GetKnownTitles`）合併 `titles` 主表 + records/purchases 推測，但 `titles` 表只要維護得夠完整，理論上就是最可信的單一來源。

**Group 安全性**：`titles` 唯一鍵是 `(group, single_number)` 複合鍵，不是單純 `single_number`——三個團體都有自己的「第 N 張單曲」編號，若只用 single_number 當鍵，不同團體同號的單曲會互相覆蓋。對應地：
- `loadTitleMap()` 回傳 `map[titleKey]string`（`titleKey{Group, SingleNumber}`），所有寫入路徑（PushRecords、UpdateTitles、PushPurchases）查表時都要帶 group
- `FixSingleTitle`／`BulkSetTitles` 的 SQL 更新條件都加上 `"group" = ?`，避免修正一個團體時誤改到另一個團體同號的記錄
- **為了讓購入花費也能做到 group 安全**，`purchases` 表補上了 `group` 欄位（原本加 group 功能時遺漏，只有 records/full_records/sign_events 有）；擴充功能的 `scrapeEntryListPage`（購入列表頁）現在也會比照個握的方式從文字判斷 group
- 既有舊資料（改名/補 group 欄位之前建立的）group 一律是空字串，不會跟任何真實 group 撞鍵，等同失效，需要透過 `/admin/titles` 重新登記

**單曲 vs 專輯的識別方式不同**：專輯沒有可靠的編號可用，擴充功能解析到的 `single_number` 一律是 `0`（即使名稱裡有「3rdアルバム」這種序號，目前也沒有存進結構化欄位）。所以：
- 單曲（`single_number > 0`）：用 `(group, single_number)` 當鍵，`titles` 表正常運作
- 專輯（`single_number == 0`）：**不會**寫入 `titles`（`FixSingleTitle`/`BulkSetTitles` 都有 `if SingleNumber != 0` 的判斷），因為用 0 當鍵會讓同團體不同專輯的名稱互相覆蓋、甚至被誤套用到下一張未公布的專輯；專輯的 タイトル未定 只能透過問題列表逐筆手動修正（且修正只影響當下符合條件的紀錄，不會留存對照表）
- `GetKnownTitles`（已知單曲名稱瀏覽）對單曲用 `(group, single_number)` 分組，對專輯改用 `(group, single_name)` 分組——名稱本身就是專輯唯一可靠的識別方式，這樣同團體的多張專輯才會各自列出，不會互相覆蓋
- `FixSingleTitle`/`BulkSetTitles` 的更新 WHERE 條件也分兩種：單曲用 `single_name != 新名稱`（因為 single_number 對單曲是唯一的，任何不一致都該修，不限 タイトル未定/空白）；專輯維持只抓 `single_name LIKE 'タイトル未定' OR single_name = ''`（不能用 != 新名稱，否則會把同團體其他正確命名的專輯一起改掉）
- **仍存在的限制**：如果同一團體「同時」有兩張未公布標題的專輯（兩筆都顯示一樣的 タイトル未定 placeholder 文字），目前無法透過 DB 設計區分是哪一張——這是來源資料本身的歧義（網站當下顯示的文字就完全相同），不是 key 設計能解決的問題

## Group（分組）欄位

`records`、`full_records`、`sign_events` 三張表都有 `group` 欄位（`nogizaka46` / `sakurazaka46` / `hinatazaka46`），無 DB index、無唯一限制。

- **個握**：`popup.js` 依抓到的事件文字關鍵字自動判斷（含「乃木坂46」→ `nogizaka46`，含「櫻坂46」→ `sakurazaka46`，含「日向坂46」→ `hinatazaka46`）
- **全握**：無法從頁面文字自動判斷（URL 結構需要 group），改由 popup.html 的 `#fullGroup` 下拉手動選擇，預設 `nogizaka46`
- 後端各列表/統計/刪除 API 都支援 `?group=` 篩選（SQL 用 `"group" = ?`，因為 `group` 是保留字需加雙引號）
- 前端 FullView / RecordsView / AdminMaintenanceView（刪除資料）的 group 下拉會連動篩選成員與單曲清單
- **已知缺口**：功能上線前已存在的舊記錄沒有 group 值（空字串），需要重新爬取才會補上，目前無後端 backfill migration

## 簽名會（Sign Event）

`sign_events` 資料表記錄簽名會抽選結果，與全握共用 `POST /scrape/full/push` 上傳端點，後端以 `order_id` 是否包含 `_sign` 字串分流到 `SignEvent` 或 `FullRecord`（`handlers/full_scraper.go`）。

- 口數顯示：`applied_count` 以 3 為一組，前端顯示 `Math.round(applied_count / 3)` + 「口」
- 結果只顯示中選（綠）/ 落選（紅），不顯示中選率
- `/admin/sign-events`（AdminSignEventsView.vue）可查看所有使用者資料（`GET /api/admin/sign-events`）
- **已知缺口**：目前 `extension/popup.js` 沒有任何程式碼會產生含 `_sign` 的 order_id（已全專案 grep 確認，分流到全握的 `parseFullApiResults` 一律輸出 `full:` 前綴），代表後端與前端頁面雖已實作，但實際上**沒有資料來源**，簽名會功能目前是空的，需要補上擴充功能端的爬取邏輯才能真正運作

## Impersonate Middleware

管理者可透過 `X-Impersonate-User: <userID>` header，以指定使用者的身份呼叫任何需要 JWT 的 `/api/*` 端點，方便除錯。

- 驗證：JWT email 必須等於 `ADMIN_EMAIL`，否則 header 被忽略
- 作用：將 gin context 中的 `userID` 替換為指定值，後續 handler 無感知
- 實作：`middleware/impersonate.go`，掛載於 `/api` group

---

## 管理者機制

`handlers/admin.go` 中以 JWT claim 的 email 做判斷（`checkAdmin`）。  
管理者 email 從 `ADMIN_EMAIL` 環境變數讀取，不寫在程式碼中。

**管理頁面三層保護：**
1. NavBar `v-if="isAdmin"` — 「管理 ▾」下拉不顯示（`is_admin` 由 `/api/me` 回傳）
2. Router guard `meta: { admin: true }` — 直接打網址導回 Dashboard
3. 後端 `checkAdmin()` — API 永遠是最終防線，回 403

**管理頁面拆成 5 個獨立路由（NavBar「管理 ▾」下拉），不再是單一頁面裡的摺疊面板：**
1. **`/admin/users` 使用者管理**（AdminUsersView.vue） — 使用者列表（email、name、record_count、last_scraped）+「模擬畫面」按鈕（搭配 impersonate store + X-Impersonate-User header）
2. **`/admin/maintenance` 資料維護**（AdminMaintenanceView.vue，內部仍用 el-collapse 分兩塊） — 刪除資料：下拉選使用者/資料類型（個握/全握/購入）/ group / 模式（全部、特定單曲、特定日期範圍）→ 先呼叫對應 `*/preview` 端點顯示實際會刪除的記錄（分頁）→ 確認後才呼叫 `DELETE`；抓取紀錄：最近 50 筆 ScrapeLog（使用者、類型、新增/跳過數、耗時、錯誤訊息）
3. **`/admin/titles` 單曲名稱**（AdminTitlesView.vue，內部用 el-collapse 分三塊） — タイトル未定修正：問題列表（含團體欄位）+ 可編輯建議標題 + 提交（`FixSingleTitle` 同時更新 records + purchases；單曲才會寫入 `titles` 主表，專輯不會，見上方「單曲 vs 專輯」說明）；單曲名稱批次登記：貼上多行 `group,single_number,single_name` 一次送出（`BulkSetTitles`），前端排重時單曲用 group+single_number、專輯（single_number=0）改用 group+single_name，不需要先出現問題才能登記；已知單曲名稱：依 group 篩選瀏覽目前 DB 裡所有已知單曲名稱（`GetKnownTitles`，單曲依編號分組、專輯依名稱分組，合併 `titles` 主表 + records/purchases 推測並顯示來源），唯讀，不是官方完整發行紀錄
4. **`/admin/venues` 場地管理**（AdminVenuesView.vue，內部用 el-collapse 分三塊，架構完全比照單曲名稱頁） — 缺少場地：問題列表（`GetVenueIssues`，全握実体場次 venue 空白的 (group, single_number, event_date) 組合）+ 可編輯場地 + 提交（`FixVenue` 回填既有空白紀錄，並寫入 `venues` 表供未來自動套用）；場地批次登記：貼上多行 `group,single_number,event_date,venue` 一次送出（`BulkSetVenues`），前端排重用 group+single_number+event_date；已知場地：依 group 篩選瀏覽（`GetKnownVenues`，合併 `venues` 表登記 + full_records 已有場地文字的實際資料並顯示來源），可編輯
5. **`/admin/sign-events` 簽名會紀錄**（AdminSignEventsView.vue） — 所有使用者的 SignEvent 列表（成員、單曲、日期、抽次、口數、中選/落選），分頁

## NavBar 導覽結構

`NavBar.vue` 用自訂下拉（`openMenu` ref + `toggle()` + click-outside 監聽，不用 el-dropdown，理由見下方）：

- **個握 ▾**：個握紀錄 (`/records`)、個握花費 (`/spending`)、個握分析 (`/records/analysis`，建設中)
- **全握 ▾**：全握紀錄 (`/full`)、全握花費 (`/full/spending`，建設中)、全握分析 (`/full/analysis`)、簽名會紀錄 (`/full/sign-events`)
- **管理 ▾**（僅 `is_admin` 顯示）：使用者管理 (`/admin/users`)、資料維護 (`/admin/maintenance`)、單曲名稱 (`/admin/titles`)、場地管理 (`/admin/venues`)、簽名會紀錄 (`/admin/sign-events`)
- 其他：總覽 (`/dashboard`)、同步 (`/scrape`)

## Pinia Stores（src/stores/）

| Store | 狀態 | 用途 |
|---|---|---|
| `auth.js` | token、refreshToken、user | JWT/refresh token 存 localStorage，401 自動續期 |
| `data.js` | `hasData` | `check()` 呼叫統計 API 判斷使用者是否已有資料，由 router guard 在 `DATA_ROUTES`（Dashboard/Records/Spending）進入時觸發 |
| `impersonate.js` | `user` | 管理者「模擬畫面」開關，`start(user)`/`stop()`，搭配 `X-Impersonate-User` header |
| `theme.js` | `isDark`、`currentMember` | 深色模式切換 + 依成員的主題色 |

## 維護模式（Maintenance Mode）

`App.vue` 啟動時 fetch `/status.json`（開發環境讀本地檔，正式環境讀 GitHub raw），若 `maintenance === true` 則條件渲染 `MaintenanceView.vue` 顯示 `message`，取代整個畫面。不是 router guard，是在 App.vue 模板層擋下。手動編輯 `status.json` 即可開關，不需重新部署。

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

- **樹狀結構**：團體 → 單曲/專輯 → 抽次 → 成員，四層 el-collapse 展開（`GetPurchaseTree` 回傳 `[]treeGroup`，每個 group 底下才是原本的 single→round→member）
- **排序**：團體跟單曲/專輯都用各自最早 `applied_at`（購入申請時間）DESC 排列，新的在前；同一團體內專輯與單曲自然依時間交錯
- **抽次排序**：`lottery_round`（整數）ASC，確保 1抽、2抽、12抽 等正確數字順序
- **formatRound**：直接用整數值顯示，如 `3` → `"3抽"`
- **依單曲佔比圓餅圖**：用 `tree.flatMap(g => g.singles)` 把所有團體的單曲攤平後再統計，圖本身不分團體

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
27. ScrapeView 改為一鍵連結 → 不再顯示 ScrapeToken 讓使用者手動複製，改為點按鈕直接透過 chrome.runtime.sendMessage 自動完成設定；擴充功能未安裝時顯示安裝步驟引導
28. 新增 Impersonate Middleware → 管理者可透過 X-Impersonate-User header 以任意使用者身份呼叫 API，便於除錯
29. 新增 title_corrections 對照表 → FixSingleTitle 修正後寫入 DB，GetTitleIssues 可優先顯示已知建議名稱；同時修正個握與購入兩張表
30. /full 路由改為 redirect 至 /dashboard → FullView.vue 保留但不再獨立路由
31. 擴充功能下載改為 GitHub URL → 從 Google Drive 改為 https://github.com/MaoIsCute/FortuneMusic/raw/main/FTExtension.zip
32. Extension 抓取流量限制修正 → fortunemusic.jp 同時只允許 1 個請求；CONCURRENCY 改為 1，批次間 500ms；偵測「アクセスが集中」頁面自動等 4 秒重試；列表頁也加入限流偵測
33. Extension 新增驗證完整性功能 → 個握與花費各新增「驗證完整性」按鈕：掃全部列表頁收集所有訂單/購入 ID，一次送 check-orders/check-entries 比對 DB，顯示缺漏數；缺漏時出現「補抓遺漏」按鈕補抓
34. Extension 新增抓取計時器 → 抓取中 progressSection 即時顯示「已執行 MM:SS」；操作記錄顯示「耗時 MM:SS」
35. ScrapeLog 新增 duration_sec 欄位 → 後端 model/handler 同步新增；AdminView 抓取記錄表格加「時長」欄位
36. 修正 JWT 並發 401 race condition → 多個請求同時收到 401 時，只有第一個執行 refresh，其餘排隊等待（pendingRequests）；refresh 成功後全部重試，失敗才統一登出
37. AdminView 名稱統一 → 刪除功能下拉「購入」改為「花費記錄」，與前端其他頁面用詞一致
38. 全握同步 URL 改為起始單頁面 → fullSyncBtn 改為開啟 ticket.fortunemeets.app/{group}/{N}{suffix}#/history，優先用起始單、次選結束單、預設 1；解決 lscache-id 在首頁讀不到的問題
39. 全握抓取改用 executeScript → fullScrapeBtn 改為在 ticket.fortunemeets.app tab 內執行 fetch，使用瀏覽器 session cookie；InternalFailureException 500 視為無資料繼續，不中止
40. 全握成員多人支援 → parseFullApiResults 改為 members 陣列全部 join「・」；GetFullRecords 篩選改為 LIKE '%name%'；FullView memberList 拆「・」去重
41. FullView 新增場地/單曲下拉篩選 → 場地從 byType 動態提取，單曲呼叫 getFullStatsBySingle；選「線上」時場地自動清空並 disabled
42. FullView 詳細紀錄改版 → 結果欄改為中選數+中選率並存，中選率套用 rateClass 三段顏色
43. AdminView 與 FullView 改為折疊式面板 → 使用 el-collapse，各區塊可展開/收合；背景灰色（#f5f7fa），卡片白色圓角陰影
44. 全站「花費」改為「個握花費」→ NavBar、SpendingView、AdminView、popup.html、popup.js 共 11 處統一更名
45. 全握新增成員詳細分析 → 後端 GET /api/full/stats/detail（group by member/venue/single/session/round）；前端選成員+場地，勾選抽次，表格動態生成欄位（部數×抽次），格內顯示中選率與中/應數
46. Supabase 連線字串更新 → 舊格式 db.xxx.supabase.co 已失效，改為新 pooler 格式 aws-1-ap-northeast-1.pooler.supabase.com:5432
47. PushRecords / PushFullRecords / PushPurchases 的 N+1 查詢 → 三者都改成先用 `IN` 批次查出已存在的 source_url/order_id/item_key 建 set，迴圈內只比對記憶體 map，最後用 GORM 批次 `Create(&slice)` 一次寫入；同批重複 key 用 map 去重避免觸發唯一索引衝突
48. 購入花費同訂單內重複列項漏算 → `fetchEntryDetailItems` 的 `fetchSingle()` 改為比照個握的合併邏輯，同 entry 內 member+date+session 相同的列項先加總再送出；另外新增小計比對驗證，抓取當下若解析總額跟頁面顯示的小計不符會在操作記錄顯示警告（僅對之後的抓取有效，不會回頭修正已存的舊資料）
49. title_corrections 跨團體撞鍵風險 → 唯一鍵改成 `(group, single_number)` 複合鍵（原本只用 single_number，三個團體號數重疊時會互相覆蓋）；`purchases` 表補上原本遺漏的 `group` 欄位；新增 `POST /api/admin/titles/bulk` 可一次登記多筆已知單曲名稱、不需等系統偵測到問題；AdminView タイトル未定修正面板加上批次貼上區塊與團體欄位顯示
50. 管理頁面從單一 AdminView.vue（5 個摺疊面板）拆成 4 個獨立路由，NavBar 改為「管理 ▾」下拉（跟個握/全握下拉同一套樣式）→ `/admin/users`（使用者管理）、`/admin/maintenance`（刪除資料 + 抓取紀錄，原因：兩者都是診斷/清理使用者資料異常時會一起用到的維運工具）、`/admin/titles`（タイトル未定修正 + 批次登記，原因：同一份 title_corrections 資料的被動修正/主動登記兩個入口）、`/admin/sign-events`（簽名會紀錄）；舊的 `/admin` 路徑 redirect 到 `/admin/users`；批次登記前端同時補上依 group+single_number 排重（取最後一筆）
51. `/admin/titles` 改名為「單曲名稱」（NavBar 顯示文字 + 頁面標題），原因：範圍已不只是 タイトル未定 修正；新增 `GET /api/admin/titles`（`GetKnownTitles`）+ 頁面內「已知單曲名稱」面板，依 group 篩選瀏覽目前 DB 裡所有已知單曲名稱與來源（已登記修正 / 個握推測 / 購入推測），唯讀
52. 專輯（single_number=0）跟單曲共用編號會互相覆蓋 → `GetKnownTitles` 改為單曲依 (group, single_number) 分組、專輯改依 (group, single_name) 分組；`FixSingleTitle`/`BulkSetTitles` 都加上 `single_number != 0` 才寫入 title_corrections 的判斷（專輯沒有可靠編號，寫入會讓不同專輯互相覆蓋、甚至誤套用到下一張未公布的專輯）；前端批次貼上的排重邏輯同步區分單曲/專輯
53. `title_corrections` 改名為 `titles`（model `TitleCorrection`→`Title`，檔名 `title_correction.go`→`title.go`，型別 `titleCorrectionKey`→`titleKey`，函式 `loadCorrectionMap`→`loadTitleMap`），定位從「被動修正對照表」改成「主動維護的單曲名稱主表」；`db.go` 新增一次性 migration 把舊表/舊索引 rename 過去（保留既有資料，不是建新空表）；AdminMaintenanceView/AdminTitlesView/AdminUsersView/AdminSignEventsView 的表格欄位寬度全部從固定 `width` 改成 `min-width`，並在 `<el-table>` 加上 `table-layout="auto"`，讓欄位依內容寬度撐開、只有真的放不下時才換行，不再強制橫向捲動
54. 表格欄位寬度即使用 `min-width` 仍可能換行 → 原因是 Element Plus 的 `min-width` 是「無設 width 欄位之間按比例分配剩餘空間的權重」，不是真正的最小寬度下限；短內容欄位（團體、單曲號）數值設太大、長文字欄位（單曲名稱、目前標題、狀態）沒設等於分到最少空間，導致比例失衡。修正：短欄位縮小到接近實際需要、長文字欄位給足夠大的 min-width；另外加 `:deep(.el-table .cell) { white-space: nowrap; }`，因為中日文字沒有空格、瀏覽器預設會優先犧牲它來換行省空間
55. 個握花費「依單曲」樹狀圖最外層加上團體分組（團體→單曲/專輯→抽次→成員）→ `GetPurchaseTree` 回傳結構改成 `[]treeGroup`，每層排序都沿用「依最早 applied_at DESC」邏輯；SpendingView 的依單曲佔比圓餅圖改用 `tree.flatMap(g => g.singles)` 把各團體攤平後再統計
56. FullAnalysisView 類型分析的「場地」欄位換行 → 原因是文件對 venue 的描述是錯的（並非簡化成東京/地方，而是原始場地全名，見上方 venue 說明修正），全名可能很長；改成 `table-layout="auto"` + 場地欄位 min-width 拉大到 200 + `:deep(.el-table .cell) { white-space: nowrap; }`，跟 admin 系列頁面同一套做法
57. 個握花費出現空白 single_name，admin 頁面無法修正 → 原因是 `GetTitleIssues`/`FixSingleTitle`/`BulkSetTitles` 的 SQL 條件都只匹配 `single_name LIKE '%タイトル未定%'`，空字串不算「含有」這個字串，所以完全不會被偵測成問題、也不會被批次/單筆修正動到；三處都改成 `single_name LIKE '%タイトル未定%' OR single_name = ''`；`PushRecords`/`UpdateTitles`/`PushPurchases` 的自動套用判斷也從「只比對 タイトル未定」改成「空字串或 タイトル未定都觸發」；AdminTitlesView 問題列表的「目前標題」欄位空白時顯示「（空白）」避免看起來像沒有問題。
58. 購入花費空白 single_name 的根本原因 → `fetchEntryDetailItems` 的 `buildSingleName()` 解析失敗（商品名稱不含「シングル」或「アルバム」字眼，如 DVD/Blu-ray、寫真集發售紀念個握）時直接 `return ''`，沒有 fallback；個握那邊的對應函式 `fetchOrderDetails` 的 `buildEventLabel()` 早就有 fallback（回傳 `parsed.event_name` 原始文字），購入這邊少做了同一件事。修正：`buildSingleName()` 解析失敗時 fallback 回傳 `info.description`（`scrapeEntryListPage` 已經抓好的原始商品名稱文字），不再是空字串
59. 問題列表只抓 タイトル未定/空白，沒有跟 `titles` 主表比對 → `GetTitleIssues` 改成：單曲（single_number > 0）如果已經登記在 `titles` 裡，任何跟登記值不一樣的名稱（含打錯字、舊版未同步等，不只 タイトル未定/空白）都算問題；還沒登記過的單曲（跟所有專輯）才退回舊版邏輯，只抓 タイトル未定/空白。同步擴大 `FixSingleTitle`/`BulkSetTitles` 的更新條件：單曲改成「跟新名稱不一樣就更新」（因為 single_number 對單曲是唯一的，任何不一致都該修），專輯維持原本只抓 タイトル未定/空白（因為同團體多張專輯共用 single_number=0，不能用「!= 新名稱」否則會誤改到別的專輯）
60. 新增簽名會個人頁面與管理者刪除功能 → 新增 `GET /api/sign-events`（個人簽名會列表）、`GET /api/admin/users/:id/sign-events/preview`、`DELETE /api/admin/users/:id/sign-events`；前端新增 `FullSignEventsView.vue`（`/full/sign-events`），NavBar「全握 ▾」加入「簽名會紀錄」；AdminMaintenanceView 加入「簽名會」刪除類型，複用 `applyDeleteFilters`
61. 擴充功能全握乃木坂46 最小起始單固定 25、預設結束 42 → 新增 `GROUP_MIN_SINGLE`/`GROUP_DEFAULT_END` 常數；`applyGroupConstraints()` 選乃木坂時自動填入限制並顯示提示訊息；起始/結束欄驗證（結束 < 起始時阻擋並提示）
62. 擴充功能錯誤訊息分層 → 使用者看到友善中文提示，後端 scrape_log 記錄技術細節（`techError || errorMsg` 雙變數模式，各 handler 分別 split 五種錯誤情境）
63. 全握成員詳細分析表格改版 → session/round 改為兩層巢狀 header（上：部數、下：抽次）；`venue` 加入 `GetFullDetailStats` GROUP BY，同張單不同場地各自一列顯示；新增「場地」欄位；預設只勾「1抽」、預設類型「実体」；`table-layout="auto"` 防止場地名被截斷
64. 全握 PushFullRecords UPDATE path 補齊 single_name → 既有記錄 `single_name` 為空字串時，重新抓取會自動補值；前端成員詳細分析「單曲」欄加 `|| n + '單'` fallback，避免舊資料空白
65. 擴充功能全握新增日向坂46 最小起始單限制 → `GROUP_MIN_SINGLE`/`GROUP_DEFAULT_END` 加入 `hinatazaka46: 4`（最小起始單 4、預設結束 17），跟乃木坂46（25/42）同一套機制；原本 `fullGroupHint` 提示文字寫死乃木坂46字樣，順便改成依目前選取團體動態產生（團體名稱 + 最小值），`popup.html` 對應改成空白容器
66. 擴充功能全握新增櫻坂46 最小起始單限制 → `GROUP_MIN_SINGLE`/`GROUP_DEFAULT_END` 加入 `sakurazaka46: 4`（最小起始單 4、預設結束 15），三團體（乃木坂/日向坂/櫻坂）目前都各自套用同一套 `applyGroupConstraints()` 機制
67. 修正 `applyGroupConstraints()` 切換團體時起始/結束單數字沒有跟著換 → 原本邏輯是「只在目前值為空或小於新團體最小值時才覆蓋」，導致從限制較高的團體（如乃木坂46 min=25）切到限制較低的團體（如日向坂46 min=4）時，因為 25 不小於 4，數字框完全沒被重設，只有下方提示文字換了；改成呼叫這個函式時（僅 popup 初始化與團體下拉切換兩處會呼叫，不會誤蓋使用者手動輸入）直接強制覆蓋成新團體的最小值/預設結束值
68. FullAnalysisView 成員統計加上團體下拉篩選 → 跟 FullView 的團體下拉同一套樣式（`GROUP_COLORS` 上色選項），`getFullStatsByMember` 後端本來就支援 `?group=` 篩選（只是前端沒傳），改為切換團體時連同類型/場地一起帶入查詢；成員欄改用 `row.group` 上對應顏色（原本沒有 `group` 欄位顯示上色，只有純文字）
69. 全握新增「地區分析」（關東場 vs 地方場） → 実体場地雖然文字各不相同（幕張メッセ／パシフィコ横浜／京都パルスプラザ／ポートメッセなごや...），但同屬關東的場地彼此中選率相近、地方場又整體跟關東不同，原本「類型分析」表格每個場地各自一列看不出這種區域級的差異。新增後端 `GET /api/full/stats/by-region`（`handlers/full_stats.go` 的 `GetFullStatsByRegion`），用 SQL `CASE WHEN venue IN (...)` 把場地即時分類成 関東/地方/その他（不動 DB schema、不需要 migration，舊資料也直接套用），只統計 `event_type = '実体'`；分類清單目前寫死在 `kantoVenues`/`regionalVenues`（`幕張メッセ`/`パシフィコ横浜`/`東京` → 関東，`京都パルスプラザ`/`ポートメッセなごや`/`地方` → 地方，涵蓋目前 DB 裡出現過的全部場地），**之後有新場地上線要手動補進這兩個清單，不會自動判斷**，沒登記到的場地會顯示「その他」；前端 `FullAnalysisView.vue` 新增「地區分析」摺疊區塊（`REGION_COLORS` 上色：関東藍／地方橘／その他灰），有自己的團體下拉篩選，跟成員統計同一套模式
70. 實測發現「その他」的 184 筆並非分類邏輯漏掉場地，而是 `event_type = '実体'` 的紀錄裡 `venue` 本身就是空字串（比任一個有名字的場地都多）——早期抓取版本還沒解析場地欄位留下的舊資料缺口，無法回頭反推屬於哪個場地/地區，只能維持在「その他」
71. 成員詳細分析（`detailType`/`detailVenue`）新增「地區」二段篩選 → 選実体後先選関東/地方（`detailRegion`），場地下拉（`detailVenueOptions`）改為只列出該地區底下的場地，選線上時地區/場地一併清空停用；後端 `GetFullDetailStats` 新增 `?region=` 參數，用同一個 `venueRegionCaseSQL()` 在 WHERE 直接比對；前端新增 `REGION_VENUES` 常數（`幕張メッセ`/`パシフィコ横浜`/`東京` → 関東、`京都パルスプラザ`/`ポートメッセなごや`/`地方` → 地方），純粹是場地下拉篩選用，內容需跟後端 `kantoVenues`/`regionalVenues` 保持一致，兩處新場地都要同步補
72. 成員統計新增「單人列/多人列」勾選（`memberRowModes`，預設兩個都勾） → 全握 `member_name` 欄位多人合場時用「・」連接多個成員名，單人跟多人合場的中選率性質不同，混在同一張表看不出差異；純前端 `filteredMemberStats` computed 依 `member_name.includes('・')` 過濾，不需要改後端（跟「抽次」篩選同一套 `el-checkbox-group` 模式，可自由組合而非單選）
73. 成員詳細分析新增「只顯示現役成員」勾選（`detailActiveOnly`） → 跟 DashboardView 既有的「在籍成員」按鈕同一套邏輯，透過 `utils/members.js` 的 `getMemberInfo(name).active ?? true` 過濾 `detailMemberOptions`（不在 MEMBERS 對照表裡的成員視為在籍，不被過濾）；勾選時若目前選取的成員被篩掉會自動清空選擇並重新查詢
74. 成員詳細分析「選擇成員」下拉排序維持團體→期別→五十音（`sortMembersByGroupAndGen`）→ 中途一度改成純期別＋五十音（忽略團體），使用者確認後還是要先分團體，改回原本寫法
75. 全握実体場次新增「場地」人工登記機制（補早期抓取版本沒有解析場地欄位的舊資料缺口，來源網站也無法回溯） → 比照既有 `titles` 單曲名稱主表的做法，新增對稱的 `models.Venue`（表 `venues`，`(group, single_number, event_date)` 複合唯一鍵 → `venue_name`；用日期而非只用單曲號當 key，因為同一張單可能跨多個日期在不同場地舉辦，如乃木坂35th單就有 5/6 與 5/26 兩個不同場次）；後端新增 `loadVenueMap()`、`GET /api/admin/venue-issues`（列出 venue 空白的組合+建議場地）、`PUT /api/admin/venue`（登記+立即回填既有空白紀錄）；`PushFullRecords`（`handlers/full_scraper.go`）在新增與更新兩個路徑都會查這張表，実体記錄 venue 空白時自動套用已登記的場地，之後同一張單同一天不管是補抓還是重新同步都會自動補上，不用每次手動修；場地功能一開始暫時放在 `/admin/titles` 頁面裡的摺疊區塊，後續（見 76）搬到獨立頁面
76. 場地管理獨立成 `/admin/venues` 頁面（AdminVenuesView.vue），不再擠在 `/admin/titles` 裡 → 架構完全比照單曲名稱頁三段式（問題列表 + 批次登記 + 已知瀏覽）：新增 `POST /api/admin/venues/bulk`（`BulkSetVenues`，批次貼上 `group,single_number,event_date,venue`，前端排重用 group+single_number+event_date 取最後一筆，同時回填既有空白紀錄）、`GET /api/admin/venues`（`GetKnownVenues`，合併 `venues` 表登記 + full_records 裡已有場地文字的實際資料，標示來源 correction/records，可編輯）；NavBar「管理 ▾」新增「場地管理」項目、router 新增 `/admin/venues`
77. 修正 `FixVenue`/`BulkSetVenues` 回填條件只認空白、無法修正已經登記錯誤的場地 → 原本 `full_records` 的 UPDATE 條件是 `venue IS NULL OR venue = ''`，同一個 (group, single_number, event_date) 第一次登記後這些紀錄就不再是空白，第二次想改成別的場地時 `venues` 對照表本身會更新成功，但 `full_records` 完全不會被 UPDATE 命中（RowsAffected=0），造成對照表跟實際紀錄不一致；改成 `venue IS NULL OR venue != ?`（跟登記值不同就更新），空白/打錯字都能修正，跟 `titles` 表對單曲的修正條件（`single_name != ?`）同一套邏輯
78. 修正「已知場地」（`GetKnownVenues`）裡來源為 `correction`（手動登記）的列單曲名稱永遠空白 → 原因是 `models.Venue`（`venues` 表）本身沒有存 `single_name` 欄位，組 `KnownVenue` 時完全沒有帶入這個值；改成額外查一次 `full_records` 裡同 (group, single_number) 的 `single_name`（不需要日期完全一致，同張單名稱都一樣）補上，只有來源 `records` 的列才會正常顯示的問題就此修正
79. `GetTitleIssues`（タイトル未定修正問題列表）新增偵測「同一單曲號出現多種互相衝突的正常名稱」→ 原本邏輯：單曲還沒登記進 `titles` 時，只有 `single_name` 是空白或含 タイトル未定 才算問題，如果 `records`/`purchases` 裡同一個 (group, single_number) 剛好有兩種都「看起來正常」（都不是空白/タイトル未定，例如打字不一致）的不同名稱，兩個都不會被判定成問題，SQL `MAX(single_name)` 在「已知單曲名稱」畫面還會悄悄吃掉其中一個，完全看不出資料兜不起來；改成先依 (group, single_number) 彙總每個不同 `single_name` 的出現次數，單曲（single_number > 0）若同時有 2 種以上非空白/非タイトル未定的名稱，全部列成獨立的問題列讓管理者比對挑出正確版本（`FixSingleTitle` 的更新條件本來就是 `single_name != 新名稱`，修其中一列會連帶把其他衝突變體一起改掉，不用逐列修）；專輯（single_number = 0）不適用這個檢查，因為同團體多張專輯本來就共用 0 這個編號，多種名稱是正常現象不是衝突
80. 單曲名稱批次登記、場地批次登記都加上「複製範例」按鈕 → 原本場地那邊的範例是純 `<pre>` + `user-select: all`（點一下自動全選再手動 Ctrl+C），單曲名稱那邊甚至只有 placeholder（無法選取複製）；兩邊統一改成 `<pre class="example-block">` 顯示範例文字 + 按鈕呼叫 `navigator.clipboard.writeText()` 直接複製到剪貼簿，成功/失敗都有 ElMessage 提示（複製失敗時提示改用手動選取，仍保留 `user-select: all` 當備援）
81. `GetVenueIssues`（缺少場地問題列表）新增偵測「場地衝突」，跟 #79 單曲名稱衝突偵測同一套邏輯 → 原本只抓 `venue IS NULL OR venue = ''`（純空白），完全偵測不到同一 (group, single_number, event_date) 底下已有非空白場地文字、但彼此不一致的情況（`PushFullRecords` 只在既有場地為空白時才會套用 `venues` 登記值，見 `full_scraper.go`，非空白但跟登記值不同的文字永遠不會被自動覆蓋）；改成分兩種情況：(1) 該 key 已登記在 `venues` 主表時，任何跟登記值不同的非空白場地文字都算問題（`FixVenue` 的更新條件本來就是 `venue != 登記值`，修正時會一併覆蓋）；(2) 該 key 還沒登記過時，只有同時出現 2 種以上不同的非空白場地文字才算問題（無法判斷哪個正確，全部列出讓管理者比對，避免把「同團體不同單曲/日期本來就该有不同場地」誤判成問題）；`VenueIssue` 新增 `current_venue` 欄位顯示目前衝突/空白的場地文字，前端「缺少場地」面板改名「缺少/衝突場地」並新增「目前場地」欄位（空白時顯示「（空白）」）
82. AdminTitlesView/AdminVenuesView 的「問題列表」「已知單曲名稱」「缺少/衝突場地」「已知場地」四個表格改成整列文字套用團體色（原本只有部分欄位單獨上色）→ 各自的 `<el-table>` 加上 `:row-style="rowStyle"`，`rowStyle({ row })` 回傳 `{ color: GROUP_COLORS[row.group] }`；因為 Element Plus 沒有在 `.el-table__cell`/`.cell` 上寫死文字顏色（只在 `.el-table` 最外層設一次），`<tr>` 上的 `color` 會正常繼承給整列文字，「（空白）」「發售日」等本來就有明確灰色樣式的次要文字不受影響，仍維持灰色好辨識
83. `kantoVenues`/`regionalVenues`（`backend/handlers/full_stats.go`）新增場地 → 関東加入 `東京ビッグサイト`，地方加入 `インテックス大阪`；前端 `FullAnalysisView.vue` 的 `REGION_VENUES` 常數（成員詳細分析場地下拉篩選用）同步補上，維持跟後端一致（見 #69/#71 的維護規則：新場地上線要手動補進這兩處，不會自動判斷）
84. 成員統計新增「地區（関東/地方）」下拉篩選，跟「成員詳細分析」的 detailRegion 同一套模式 → 後端 `GetFullStatsByMember` 新增 `?region=` 參數（`venueRegionCaseSQL() = ?`，跟 `GetFullDetailStats`/`GetFullStatsByRegion` 共用同一個 SQL CASE 判斷式）；前端新增 `memberFilterRegion`，場地下拉改用 `memberVenueOptions`（依所選地區從 `venueList` 篩出對應場地，跟 `detailVenueOptions` 同一套邏輯），切到「線上」類型或切換地區時会自動清空不相容的地區/場地選擇再重新查詢
85. 成員詳細分析新增「團體」下拉篩選（純前端，跟成員統計/地區分析同一套 GROUP_COLORS 三團體下拉樣式）→ 新增 `detailFilterGroup`，`detailMemberOptions` 改成同時比對 group + 現役兩個條件（原本只檢查 `detailActiveOnly`）；不需要改後端，因為這裡只是縮小「選擇成員」下拉的選項範圍，實際查詢仍是靠 `GetFullDetailStats` 的 `member` LIKE 參數比對選到的單一成員名稱；切換團體時若目前選取的成員不在新篩選結果內，比照現役成員 checkbox 的既有邏輯（`onDetailActiveOnlyChange`）自動清空選擇並重新查詢
86. 「總覽」的統計內容（全體統計卡、各次應募中選率折線圖、各部中選率長條圖、各筆應募中選率長條圖、成員手風琴列表）整個搬到「個握分析」（`/records/analysis`，原本只是「建設中」佔位頁）→ `RecordsAnalysisView.vue` 取代 `DashboardView.vue` 原本的完整內容（含 script/style），只把標題從「總覽」改成「個握分析」；`router/index.js` 的 `DATA_ROUTES` 加入 `RecordsAnalysis`，讓它比照 Records/Spending 在進站前先做資料檢查。（這條後續被 #87 取代：`/dashboard` 不再是純空殼，改造成「全員統計」頁面，原本這條加的 onboarding 提示卡邏輯已被移除，改由 LockedState 元件的「前往同步工具」按鈕引導新使用者，`/setup` 的自動偵測導向目前只保留在 `RecordsAnalysisView.vue`）
87. 新增「全員統計」（`/dashboard`，NavBar「總覽」改名「全員統計」）→ 跨使用者聚合的個握／全握中選率分析，分兩個各自獨立解鎖的區塊：
    - **後端**新增 `backend/handlers/global_stats.go`：個握聚合端點（`GetGlobalOverallStats`/`GetGlobalDetailStats`/`GetGlobalOrderSequenceStats`）跟個人版 `stats.go` 同一套查詢，差別只在拿掉 `user_id = ?` 篩選；`GetGlobalOrderSequenceStats` 的 `position` 因此變成「該場次全員送出順序」而不是單一使用者自己的順序，樣本數更大更有參考價值。另外新增兩個個人版沒有對應端點的「排行榜」——`GetGlobalStatsByMember`（依成員）、`GetGlobalStatsBySingle`（依單曲，join `titles` 補發售日），因為單一使用者抽的成員/單曲數太少，這種排行只有聚合後才有意義。全握比照新增 `GetGlobalFullOverallStats`/`GetGlobalFullStatsByMember`/`GetGlobalFullStatsByRegion`/`GetGlobalFullDetailStats`，同樣是拿掉 `user_id` 篩選版本的 `full_stats.go`。兩個 overall 端點都加了 `contributor_count`（`SELECT DISTINCT user_id` 計數），供頁面顯示「貢獻人數」。刻意決定：**顆粒度跟個人版完全一致（含日期/場次/場地），不做人數門檻限制**，因為這個工具的目的是「用最少資源拿最高中選率」，日期/場次級的歷史中選率正是最實用的資訊，砍掉細節反而喪失功能價值；**簽名會（SignEvent）完全不在聚合範圍內**，因為牽涉見面資格較敏感。解鎖判斷純前端邏輯，後端不額外加「有沒有貢獻」的檢查（這是鼓勵貢獻的軟性誘因，不是安全邊界）。路由掛在 `/api/global/...`，一樣走 JWT 驗證。
    - **前端** `DashboardView.vue` 整個重寫：`recordsUnlocked` 沿用既有 `dataStore.hasData`（router 進站前已經 `check()` 過，等於「該帳號個握 total_applied > 0」，剛好就是個握總表的解鎖條件）；`fullUnlocked` 另外呼叫個人版 `getFullOverallStats()` 檢查 `total_applied > 0`（沒有現成 store，直接呼叫既有個人端點）。未解鎖時顯示新元件 `components/LockedState.vue`（圖示＋標題＋說明＋「前往同步工具」按鈕導去 `/scrape`，外觀比照既有 `EmptyState.vue` 但多一個 CTA，個握/全握兩處共用同一個元件、傳不同文字）。個握總表的圖表/手風琴邏輯整段從舊版 `DashboardView.vue`（見 #86）搬回來，改接全員端點，並新增「排行榜」區塊（依成員/依單曲，共用一個團體篩選）；全握總表整段比照 `FullAnalysisView.vue` 的類型分析/地區分析/成員統計/成員詳細分析四個摺疊區塊，改接全員端點。因為兩塊邏輯合併在同一個檔案裡，所有 state 變數依區塊加上 `r`/`f` 字首前綴（`rOverall`/`fOverall`、`rRows`/`fMemberStats` 等）避免同名衝突，純工具函式（`rateClass`/`formatSingle`/`groupLabel`）跟兩塊都會用到的就只定義一份不重複。
    - **連帶修正**：`ScrapeView.vue` 「抓取完成後，點左上角「總覽」就能看到你的統計資料了」這句提示改成「個握 ▾ → 個握分析」（因為 #86 已經把個人統計搬到個握分析，總覽現在顯示的是全員聚合，不是自己的資料）。
88. 「全員統計」的個握總表/全握總表從 `el-collapse` 兩個摺疊區塊改成 `el-tabs` 兩個頁簽（`activeTab`，預設停在「個握總表」），區塊內部結構完全不變，只是外層容器從 collapse-item 換成 tab-pane。頁面最上方新增一個可收合的「🚀 新手上路：快速上手指南」區塊（`quickStartOpen`，四步驟：連結同步工具 → 開始同步資料 → 查看自己的統計 → 解鎖全員統計，第一步附「前往同步工具」連結），右下角有「不再顯示」，點了寫 localStorage（`dashboardQuickStartDismissed`）記住，下次進站預設收合；沒點過的話預設展開。因為 `/dashboard` 是登入後預設頁，這個指南放在使用者第一眼會看到的位置，比另外做一個獨立首頁路由更直接。
89. 「各次應募中選率比較」折線圖（`RecordsAnalysisView.vue` 個握分析 + `DashboardView.vue` 個握總表，兩處實作一致）新增「團體」「成員」兩個下拉篩選 → 原本只能點圖表下方 echarts 內建 legend 逐一切換成員顯示/隱藏，成員一多（最多 87 人）很難在密密麻麻的 legend 裡找到特定人；新增 `chartFilterGroup`（多選，narrow「成員」下拉的選項範圍）+ `chartFilterMembers`（多選、可搜尋，實際決定折線圖顯示誰），成員下拉不選任何人時維持原本「全部顯示」的預設行為，選了才只顯示被選中的成員；`legendSelected` 沿用原本機制不變，下拉只是批次改寫它，使用者選完後仍可以直接點 legend 做微調（不是取代原本的點擊互動，是疊加一層更快速的篩選入口）；`watch(memberMap, ...)` 初始化 legend 時也改成尊重目前的篩選狀態，不會每次資料變動就重置回全選。DashboardView 版本沿用既有的 `r` 字首命名慣例（`rChartFilterGroup`/`rChartFilterMembers`/`rChartMemberOptions`/`rOnChartGroupChange`/`rApplyChartFilter`）。
90. 修正個握場次「第1部」跟「1部」被當成兩個不同場次分開統計（各部中選率、成員手風琴部數分層都會拆開） → 根因是 `extension/popup.js` 解析場次的正規表達式 `(第?\d+部)` 把「第」設成可有可無，直接照原始網頁文字存進 `session` 欄位，而 fortunemusic 網站在不同頁面（申請確認頁 vs 訂單明細頁）顯示場次的方式本身就不一致；修正：新增 `normalizeSession()` 統一補上「第」前綴，套用在 `fetchOrderDetails`（個握）跟 `fetchEntryDetailItems`（花費）兩處捕捉 session 的地方，全握的 session 來自網站自己的 JSON API 欄位（`prizeInfo.part`），不是這個正規表達式，不受影響；後端 `db/db.go` 新增一次性冪等 migration，把 `records`/`purchases` 兩張表裡符合 `^[0-9]+部$`（沒有「第」）的既有 session 值統一補上「第」，讓歷史資料也合併回同一場次；`extension/manifest.json` 版本號 1.7 → 1.8，`FTExtension.zip` 已重新打包。
91. 「全員統計」個握總表底下的內容（折線圖、各部中選率、各筆應募中選率、排行榜、成員手風琴列表）改成用 `el-collapse` 包起來，各自可以點標頭展開/收合，跟全握總表既有的類型分析/地區分析/成員統計/成員詳細分析四個摺疊區塊統一風格（`rOpenSections`，預設全部展開）；原本包裹每張圖的 `.chart-card` 白色卡片樣式拿掉，改吃 `el-collapse-item` 本身的卡片外觀（同一份樣式已經在用），避免卡片疊卡片；連帶清掉變成無用的 `.chart-card`/`.chart-header`/`.chart-title` CSS（含深色模式覆寫），標題文字統一改放進 `<template #title>` 用共用的 `.collapse-title`。全體統計卡（貢獻人數/總應募/總中選/總中選率）維持攤開不收合。
92. 「個握分析」（`RecordsAnalysisView.vue`）比照 #91 全員統計/個握總表的做法，把折線圖、各部中選率、各筆應募中選率、成員手風琴列表四塊也改成 `el-collapse` 包起來可各自收合（`openContentSections`，預設全部展開），這個頁面原本沒有排行榜所以只有四塊不是五塊；因為這個頁面之前完全沒用過 `el-collapse`，額外把 `:deep(.el-collapse-item)` 等卡片樣式跟 `.collapse-title` 從 DashboardView/FullAnalysisView 那套搬過來，同樣清掉不再使用的 `.chart-card`/`.chart-header`/`.chart-title`。順便確認了目前成員排序邏輯（`sortedMembers`）：先依期別（`gen`，來自 `utils/members.js` 的 `MEMBERS` 對照表，不在表裡的排最後）、期別相同再依五十音（`localeCompare('ja')`），不分團體——同期別的不同團體成員會依五十音混在一起，不會先按團體分堆。
93. 個握分析 + 全員統計/個握總表的「成員列表」手風琴最上面加一層「團體」，變成 團體 → 成員 → 單曲 → 抽次 四層（原本三層）；團體固定順序乃木坂46→櫻坂46→日向坂46，可點標頭展開/收合，標頭一併顯示該團體加總的應募/中選/中選率，跟其他層級呈現方式一致。團體內成員排序維持原本「期別→五十音」不變——`sortedMembers`/`rSortedMembers` 已經排好的順序直接按 group 分桶，桶內相對順序自然保留，不用重新排序。技術細節：`utils/members.js` 的 `MEMBERS` 靜態對照表原本沒有存團體欄位，補上 `group` 欄位（依照檔案內本來就有的團體分區塊機械式加入，共 202 筆：乃木坂46 98、櫻坂46 58、日向坂46 46）；`memberMap`/`rMemberMap` 建立成員資料時，group 優先吃 `getMemberInfo(name).group`（穩定、不受舊資料缺 group 影響），沒登記在對照表裡的成員才 fallback 用 `row.group`（可能是空字串，因為早期記錄沒有 group 欄位，這種情況會被歸類到「—」桶，見已知缺口）。
94. 修正五十音排序：`localeCompare('ja')` 對漢字排出來的不是真正的日文姓名讀音順（是 Unicode/筆畫序），例如「吉田」（よしだ）跟「与田」（よだ）都是よ開頭，實際排序卻被拆得很遠。修正分兩部分：
    - **`utils/members.js` 的 `MEMBERS` 資料本身**：203→202 筆全部人工比對真實讀音、依團體（乃木坂46→櫻坂46→日向坂46）、期別、五十音手動重新排列宣告順序（過程中一併修正了幾個期別分類錯誤：山崎怜奈 3期→2期、井上小百合/斎藤ちはる/中田花奈 2期→1期）。順便把櫻坂46／日向坂46原本「N期」跟「畢業N期」分開兩個區塊的結構，仿照乃木坂46的做法合併成同一個「N期」區塊（現役+畢業混排，只靠 `active` 欄位分辨，不靠檔案裡的區塊位置），因為排序/篩選邏輯本來就只吃 `gen`/`active` 欄位值，跟區塊沒關係，合併後才能在同一個 gen 底下正確依五十音排序。
    - **排序函式本身**：`sortMembersByGen`/`sortMembersByGroupAndGen`（`utils/members.js`）以及兩個頁面（`DashboardView.vue`/`RecordsAnalysisView.vue`）各自複製的 `rSortedMembers`/`sortedMembers` inline 排序、還有「各次應募中選率比較」折線圖 legend 的成員排序，全部把 `localeCompare('ja')` 換成新增的 `memberOrderIndex(name)`（`utils/members.js` 新增匯出，讀 `MEMBERS` 物件宣告順序建的 `name → 索引` 對照表）。因為現在 `MEMBERS` 宣告順序本身就是校對過的正確五十音順，排序時直接比索引數字，比字串比較快，也不會再排錯。
    - `sortMembersByGen`（純期別→五十音，不分團體）確認整個專案沒有任何地方呼叫，是死函式，順手刪除。
95. 移除「成員詳細分析」（`DashboardView.vue`/`FullAnalysisView.vue` 的 `fDetailRows`/`detailRows`）排序條件裡多餘的 `a.partner.localeCompare(b.partner)` tie-break → 分組用的 `key` 本身就是 `` `${single_number}:${member_name}:${venue}` ``，已經包含 `member_name`，代表同一列（同單曲+同場地）本來就只會對應到固定的一組搭檔，不會有「同一格裡搭檔不同、需要再排序決定顯示順序」的情況，所以拿掉這個沒有意義的排序條件，只保留 `single_number` 主排序 + `venue` 次排序。這也代表 #94 提到「搭檔欄位沒排序」的已知缺口其實不需要修，原本就不需要排序。
