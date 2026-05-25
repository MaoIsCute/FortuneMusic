const BACKEND_KEY = 'backendUrl'
const TOKEN_KEY   = 'scrapeToken'

const setupPage = document.getElementById('setupPage')
const mainPage  = document.getElementById('mainPage')
const statusEl  = document.getElementById('status')
const resultEl  = document.getElementById('result')
const syncBtn   = document.getElementById('syncBtn')
const scrapeBtn = document.getElementById('scrapeBtn')

// ─── 在 fortunemusic.jp/mypage/apply_list/ 執行的爬蟲 ──────────────────────
// 此函式被序列化注入分頁，不可引用外部變數。
// 假設分頁已載入完成且使用者已登入。
async function scrapeFromApplyListPage() {
  const itemRe = /^(.+?)【(\d{1,2}\/\d{1,2})\s+(第\d+部)】(.+)$/
  const textRe = /[一-鿿぀-ゟ゠-ヿ･-ﾟa-zA-Z]+【\d{1,2}\/\d{1,2}\s+第\d+部】.+/g

  function parseProductName(text) {
    const m = text.trim().match(itemRe)
    if (!m) return null
    return { member_name: m[1].trim(), raw_date: m[2], session: m[3], event_name: m[4].trim() }
  }

  // 確認是否登入（有登入表單 = 未登入）
  const hasLoginForm = !!document.querySelector('[name="login_form"], input[type="password"]')
  if (hasLoginForm) {
    return { error: '頁面顯示登入表單，請先登入後再點「開始抓取」' }
  }

  // 從目前 DOM 收集訂單 ID、應募日期、單曲資訊
  const orderInfoMap = {} // orderID → { year, month, singleNum, singleSuffix, singleTitle }
  document.querySelectorAll('a[href]').forEach(a => {
    const m = (a.getAttribute('href') || '').match(/\/mypage\/apply_detail\/(\d+)\/?/)
    if (!m || orderInfoMap[m[1]] !== undefined) return
    const id = m[1]
    orderInfoMap[id] = null // 佔位，避免重複處理

    const container = a.closest('tr, li, article, section') || a.parentElement
    if (!container) return

    const info = {}

    // 応募日時
    container.querySelectorAll('span.hdg').forEach(span => {
      if (span.textContent.trim() !== '応募日時') return
      const tdText = span.parentElement.textContent.trim()
      const dateStr = tdText.replace(span.textContent.trim(), '').trim()
      const dm = dateStr.match(/(\d{4})-(\d{1,2})-/)
      if (dm) { info.year = parseInt(dm[1]); info.month = parseInt(dm[2]) }
    })

    // 單曲號、歌名、應募次數（從 td.tdEvent 解析）
    const tdEvent = container.querySelector('td.tdEvent')
    if (tdEvent) {
      const eventText = tdEvent.textContent.trim()
      const sm = eventText.match(/(\d+)(st|nd|rd|th)シングル/)
      const tm = eventText.match(/『(.+?)』/)
      const rm = eventText.match(/第(\d+)次/)
      if (sm) {
        info.singleNum    = sm[1]
        info.singleSuffix = sm[2]
        info.singleTitle  = tm ? tm[1] : null
        info.lotteryRound = rm ? `第${rm[1]}次` : null
      }
    }

    if (Object.keys(info).length) orderInfoMap[id] = info
  })

  const orderIDs = Object.keys(orderInfoMap)
  if (orderIDs.length === 0) {
    const title = document.querySelector('title')?.textContent?.trim() || ''
    return { records: [], order_count: 0, title }
  }

  // same-origin fetch 各申請詳情
  const parser = new DOMParser()
  const records = []

  for (const id of orderIDs) {
    let res
    try { res = await fetch(`/mypage/apply_detail/${id}/`) } catch { continue }
    if (!res.ok) continue

    const detailDoc = parser.parseFromString(await res.text(), 'text/html')
    const sourceBase = `https://fortunemusic.jp/mypage/apply_detail/${id}/`
    const applyInfo  = orderInfoMap[id]

    // 同一張單內若有相同 member+date+session 的多行（部分中選時逐筆顯示）
    // 用 aggregated 累加，避免 seen 跳過後續行導致數據丟失
    const aggregated = {}

    detailDoc.querySelectorAll('tbody tr:not(.tblCatLast)').forEach(row => {
      const nameTd = row.querySelector('td:first-child')
      if (!nameTd) return

      const parsed = parseProductName(nameTd.textContent)
      if (!parsed) return

      const quaCells = row.querySelectorAll('td.tdQua')
      const applied  = parseInt((quaCells[0]?.textContent || '').match(/\d+/)?.[0] || '0')
      const won      = parseInt((quaCells[1]?.textContent || '').match(/\d+/)?.[0] || '0')

      const key = parsed.member_name + parsed.raw_date + parsed.session

      if (aggregated[key]) {
        aggregated[key].applied_count += applied
        aggregated[key].won_count     += won
      } else {
        // 用應募月推算活動年份：活動月 < 應募月 → 隔年
        const eventMonth = parseInt(parsed.raw_date.split('/')[0])
        let eventYear
        if (applyInfo) {
          eventYear = eventMonth < applyInfo.month ? applyInfo.year + 1 : applyInfo.year
        } else {
          eventYear = new Date().getFullYear()
        }

        // source_url 使用 raw_date（M/D）維持與既有資料的去重相容性
        const sourceURL = `${sourceBase}#${encodeURIComponent(parsed.member_name)}|${parsed.raw_date}|${parsed.session}`

        // 用 tdEvent 解析出的單曲資訊組成 event_name（單曲/應募次數）
        let eventLabel = parsed.event_name
        if (applyInfo?.singleNum) {
          eventLabel = `${applyInfo.singleNum}${applyInfo.singleSuffix}シングル`
          if (applyInfo.singleTitle)  eventLabel += `「${applyInfo.singleTitle}」`
          if (applyInfo.lotteryRound) eventLabel += `/${applyInfo.lotteryRound}`
        }

        aggregated[key] = {
          member_name:   parsed.member_name,
          event_date:    `${eventYear}/${parsed.raw_date}`,
          session:       parsed.session,
          event_name:    eventLabel,
          applied_count: applied,
          won_count:     won,
          source_url:    sourceURL,
        }
      }
    })

    Object.values(aggregated).forEach(rec => records.push(rec))
  }

  return { records, order_count: orderIDs.size }
}
// ────────────────────────────────────────────────────────────────────────────

async function init() {
  const data = await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])
  if (data[BACKEND_KEY] && data[TOKEN_KEY]) showMain(data[BACKEND_KEY])
  else showSetup(data[BACKEND_KEY] || '')
}

function showSetup(defaultUrl) {
  setupPage.style.display = 'block'
  mainPage.style.display  = 'none'
  document.getElementById('backendUrl').value = defaultUrl || 'http://localhost:8080'
}

function showMain(backendUrl) {
  setupPage.style.display = 'none'
  mainPage.style.display  = 'block'
  statusEl.textContent    = '已連接：' + backendUrl
  statusEl.className      = 'status connected'
}

function showResult(type, message) {
  resultEl.style.display = 'block'
  resultEl.className     = 'result ' + type
  resultEl.textContent   = message
}

function setWaitingMode(on) {
  syncBtn.style.display   = on ? 'none'  : 'block'
  scrapeBtn.style.display = on ? 'block' : 'none'
}

document.getElementById('saveBtn').addEventListener('click', async () => {
  const backendUrl  = document.getElementById('backendUrl').value.trim()
  const scrapeToken = document.getElementById('scrapeToken').value.trim()
  if (!backendUrl || !scrapeToken) { alert('請填入所有欄位'); return }
  await chrome.storage.local.set({ [BACKEND_KEY]: backendUrl, [TOKEN_KEY]: scrapeToken })
  showMain(backendUrl)
})

document.getElementById('settingsBtn').addEventListener('click', async () => {
  const data = await chrome.storage.local.get([BACKEND_KEY])
  showSetup(data[BACKEND_KEY] || '')
})

// 步驟一：開啟申請列表分頁
syncBtn.addEventListener('click', async () => {
  await chrome.tabs.create({
    url: 'https://fortunemusic.jp/mypage/apply_list/',
    active: true,
  })
  setWaitingMode(true)
  showResult('info',
    '已開啟申請列表頁面。\n\n' +
    '① 如未登入，請在分頁中登入\n' +
    '② 確認看到申請記錄列表後\n' +
    '③ 點「開始抓取」'
  )
})

// 步驟二：從已開啟的分頁抓取資料
scrapeBtn.addEventListener('click', async () => {
  const { [BACKEND_KEY]: backendUrl, [TOKEN_KEY]: scrapeToken } =
    await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])

  scrapeBtn.disabled    = true
  scrapeBtn.textContent = '抓取中...'

  try {
    // 找到 apply_list 分頁
    const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/mypage/apply_list/*' })
    if (tabs.length === 0) {
      showResult('error', '找不到申請列表分頁，請先點「同步」開啟頁面，確認登入後再試')
      return
    }

    showResult('info', '正在讀取申請記錄...')

    const results = await chrome.scripting.executeScript({
      target: { tabId: tabs[0].id },
      func: scrapeFromApplyListPage,
    })

    const r = results[0]?.result
    if (!r)       { showResult('error', '抓取失敗，無法取得結果'); return }
    if (r.error)  { showResult('error', r.error);                  return }

    const { records, order_count } = r
    if (order_count === 0) {
      showResult('warning', `申請列表是空的（頁面：${r.title || '-'}）`)
      setWaitingMode(false)
      return
    }
    if (records.length === 0) {
      showResult('warning', `找到 ${order_count} 筆申請但無法解析記錄格式\n請回報此問題`)
      return
    }

    showResult('info', `解析完成（${records.length} 筆），上傳中...`)

    const res  = await fetch(`${backendUrl}/scrape/push`, {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify({ scrape_token: scrapeToken, records }),
    })
    const json = await res.json()

    if (!res.ok) { showResult('error', json.error || '後端儲存失敗'); return }

    if (json.new_records > 0) {
      const note = json.skipped > 0 ? `（跳過重複 ${json.skipped} 筆）` : ''
      showResult('success', `同步成功！新增 ${json.new_records} 筆記錄${note}`)
    } else if (json.skipped > 0) {
      showResult('success', `同步完成，所有 ${json.skipped} 筆已存在，無新增`)
    } else {
      showResult('warning', '同步完成，沒有可儲存的記錄')
    }

    setWaitingMode(false)
  } catch (e) {
    showResult('error', '發生錯誤：' + e.message)
  } finally {
    scrapeBtn.disabled    = false
    scrapeBtn.textContent = '開始抓取'
  }
})

init()
