# Fortune Tracker — Project 說明文件

## 專案概述
統計 **Fortune Music** 網站上乃木坂46見面會抽選結果的工具。
使用者安裝 Chrome 擴充功能後，可一鍵同步自己的抽選紀錄，並在統計畫面分析中選機率。

---

## 技術選型

| 層級 | 技術 |
|------|------|
| 前端 | Vue 3 + Vue Router + Pinia + Element Plus |
| 後端 | Go + Gin + GORM |
| 資料庫 | PostgreSQL（Supabase 托管，Session pooler 連線） |
| 登入 | Google OAuth 2.0 + JWT（15min）+ Refresh Token（30天） |
| 擴充功能 | Chrome Extension Manifest V3 |
| 前端部署 | Vercel |
| 後端部署 | Render（免費方案 + UptimeRobot 每5分鐘 keep-alive） |

---

## 專案結構

```
D:\Project\fortunemusic\
├── backend\
│   ├── main.go
│   ├── .env
│   ├── config/config.go          # 環境變數設定
│   ├── db/db.go                  # PostgreSQL 連線
│   ├── models/                   # User, Record, Purchase... 資料表
│   ├── handlers/
│   │   ├── auth.go               # Google OAuth + JWT
│   │   ├── scrape.go             # 擴充功能爬蟲接收 API
│   │   ├── stats.go              # 個握統計 API
│   │   ├── full.go               # 全握統計 API
│   │   └── purchase.go           # 花費統計 API
│   ├── middleware/auth.go        # JWT 驗證
│   └── router/router.go         # 路由設定
├── frontend\
│   ├── vercel.json               # SPA routing rewrite
│   └── src/
│       ├── api/index.js          # Axios + 自動 refresh token
│       ├── router/index.js       # Vue Router（含 data guard）
│       ├── stores/
│       │   ├── auth.js           # 登入狀態
│       │   ├── theme.js          # 成員應援色主題
│       │   └── data.js           # hasData 全域快取
│       ├── components/
│       │   ├── NavBar.vue
│       │   ├── EmptyState.vue    # 無資料提示卡
│       │   └── ErrorState.vue    # 連線失敗提示卡
│       └── views/
│           ├── LoginView.vue
│           ├── AuthCallbackView.vue  # OAuth 回調（含自動連結擴充）
│           ├── SetupView.vue     # 新手安裝引導頁
│           ├── DashboardView.vue # 統計總覽 + 圖表
│           ├── RecordsView.vue   # 抽選紀錄列表
│           ├── SpendingView.vue  # 花費統計
│           ├── ScrapeView.vue    # 同步工具手動設定
│           └── AdminView.vue     # 管理員工具
└── extension\
    ├── manifest.json
    ├── background.js             # PING + FORTUNE_SETUP 訊息接收
    ├── popup.html / popup.css
    └── popup.js                  # 同步邏輯（個握、花費）
```

---

## 資料流

```
使用者 → Chrome 擴充功能（popup）
  → 開啟 fortunemusic.jp（帶 session cookie）
  → chrome.scripting.executeScript 注入爬蟲
  → 爬取申請列表 + 訂單明細
  → POST /scrape/push（帶 scrape_token）
  → 後端存入 PostgreSQL
  → 前端 /dashboard 顯示統計
```

---

## 同步工具設定流程

1. 使用者以 Google 帳號登入網站
2. `AuthCallbackView` 登入成功後自動嘗試連結擴充功能（靜默）
3. 若擴充功能未安裝 → 導向 `/setup` 顯示安裝步驟
4. 安裝完成後在 `/setup` 點「連結帳號」按鈕完成設定
5. 之後點擴充功能圖示 → 點「同步」即可

---

## API 端點

### 認證
| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | /auth/google | 導向 Google OAuth |
| GET | /auth/google/callback | OAuth callback，回傳 JWT |
| POST | /auth/refresh | 刷新 access token |

### 使用者
| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | /api/me | 取得目前使用者資訊 |
| GET | /api/scrape-token | 取得擴充功能用的 scrape_token |

### 個握統計
| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | /api/stats/overall | 整體中選率 |
| GET | /api/stats/by-member | 依成員統計 |
| GET | /api/stats/detail | 詳細資料（成員×單曲×次數×部數） |
| GET | /api/stats/order-sequence | 依訂單序號統計 |
| GET | /api/records | 抽選紀錄列表（分頁） |

### 花費統計
| 方法 | 路徑 | 說明 |
|------|------|------|
| GET | /api/purchases/stats/overall | 花費總計 |
| GET | /api/purchases/tree | 依單曲樹狀統計 |
| GET | /api/purchases/stats/by-member | 依成員花費 |

### 爬蟲接收（擴充功能呼叫）
| 方法 | 路徑 | 說明 |
|------|------|------|
| POST | /scrape/check-orders | 確認哪些訂單 ID 是新的 |
| POST | /scrape/push | 上傳個握紀錄 |
| POST | /scrape/update-titles | 更新舊訂單的單曲名稱 |
| POST | /scrape/check-entries | 確認哪些購入記錄是新的 |
| POST | /scrape/purchases/push | 上傳花費紀錄 |
| POST | /scrape/log | 記錄同步日誌 |

---

## 前端路由

| 路徑 | 頁面 | 說明 |
|------|------|------|
| / | LoginView | Google 登入頁 |
| /auth/callback | AuthCallbackView | OAuth 回調 |
| /setup | SetupView | 新手安裝引導（擴充功能未安裝時導向） |
| /dashboard | DashboardView | 統計總覽、圖表、成員手風琴 |
| /records | RecordsView | 抽選紀錄列表（可篩選） |
| /spending | SpendingView | 花費統計 |
| /scrape | ScrapeView | 同步工具手動設定 |
| /member/:name | MemberView | 成員詳細統計 |
| /admin | AdminView | 管理員（title 修正、使用者管理） |

---

## 空狀態邏輯

- **無資料 + 無擴充功能** → 導向 `/setup`
- **無資料 + 有擴充功能** → 頁面內顯示 `EmptyState` 提示卡
- **連線失敗** → 顯示 `ErrorState` 提示卡 + 重新整理按鈕
- Router guard 進入 Dashboard/Records/Spending 前先查 `dataStore.hasData`，避免頁面閃爍

---

## 環境變數

### backend/.env
```
DATABASE_URL=postgresql://postgres.[PROJECT]:[PASSWORD]@aws-1-ap-northeast-1.pooler.supabase.com:5432/postgres
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
JWT_SECRET=
REFRESH_SECRET=
APP_URL=https://fortunemusictracker.onrender.com
FRONTEND_URL=https://fortune-music-cehnyf0sw-sams-projects-f6308fd5.vercel.app
```

### frontend/.env
```
VITE_API_URL=https://fortunemusictracker.onrender.com
```

---

## 部署注意事項

- **Render 免費方案** 閒置 15 分鐘後 sleep → 用 UptimeRobot 每 5 分鐘打 `/health` 保活
- **Supabase** 使用 Session pooler URL（`pooler.supabase.com:5432`）解決 Render IPv6 問題
- **擴充功能** 採 Google Drive zip 方式分發，不上架 Chrome Web Store
- **CORS** 後端 `FRONTEND_URL` 環境變數需與 Vercel 部署 URL 一致
