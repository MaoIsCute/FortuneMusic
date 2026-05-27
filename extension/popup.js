const BACKEND_KEY = 'backendUrl'
const TOKEN_KEY   = 'scrapeToken'

const setupPage = document.getElementById('setupPage')
const mainPage  = document.getElementById('mainPage')
const statusEl  = document.getElementById('status')
const resultEl  = document.getElementById('result')
const syncBtn   = document.getElementById('syncBtn')
const scrapeBtn = document.getElementById('scrapeBtn')

// ─── 階段一：掃描申請列表，不進 detail page ──────────────────────────────────
// 回傳 { orders: [{id, info}], hasMore } 或 { error }
async function scrapeListPage(pageNum) {
  let doc
  if (pageNum === 1) {
    doc = document
    if (doc.querySelector('[name="login_form"], input[type="password"]'))
      return { error: '頁面顯示登入表單，請先登入後再點「開始抓取」' }
  } else {
    let res
    try { res = await fetch(`/mypage/apply_list/?page=${pageNum - 1}`) } catch (e) {
      return { error: `第 ${pageNum} 頁讀取失敗：${e.message}` }
    }
    if (!res.ok) return { error: `第 ${pageNum} 頁回應錯誤：${res.status}` }
    doc = new DOMParser().parseFromString(await res.text(), 'text/html')
  }

  const orders = []
  const seen   = {}

  doc.querySelectorAll('a[href]').forEach(a => {
    const m = (a.getAttribute('href') || '').match(/\/mypage\/apply_detail\/(\d+)\/?/)
    if (!m || seen[m[1]]) return
    seen[m[1]] = true
    const id  = m[1]
    const info = {}

    const container = a.closest('tr, li, article, section') || a.parentElement
    if (!container) { orders.push({ id, info: null }); return }

    container.querySelectorAll('span.hdg').forEach(span => {
      if (span.textContent.trim() !== '応募日時') return
      const tdText  = span.parentElement.textContent.trim()
      const dateStr = tdText.replace(span.textContent.trim(), '').trim()
      const dm = dateStr.match(/(\d{4})-(\d{1,2})-/)
      if (dm) { info.year = parseInt(dm[1]); info.month = parseInt(dm[2]) }
    })

    const tdEvent = container.querySelector('td.tdEvent')
    if (tdEvent) {
      const eventText = tdEvent.textContent.trim()
      const sm    = eventText.match(/(\d+)(st|nd|rd|th)シングル/)
      const am    = eventText.match(/(\d+)(st|nd|rd|th)アルバム/)
      const titleM = eventText.match(/[『「](.+?)[』」]/)
      const rm    = eventText.match(/第(\d+)次/)
      if (rm) info.lotteryRound = `第${rm[1]}次`
      if (sm) {
        info.singleNum    = sm[1]
        info.singleSuffix = sm[2]
        info.singleTitle  = titleM ? titleM[1] : null
      } else if (am) {
        info.albumNum    = am[1]
        info.albumSuffix = am[2]
        info.albumTitle  = titleM ? titleM[1] : null
      } else if (/アルバム/.test(eventText)) {
        info.isAlbum   = true
        info.albumTitle = titleM ? titleM[1] : null
      }
    }

    orders.push({ id, info: Object.keys(info).length ? info : null })
  })

  if (orders.length === 0) return { orders: [], hasMore: false }
  return { orders, hasMore: true }
}

// ─── 階段二：只抓新訂單的 detail page ────────────────────────────────────────
// entries = [{id, info}]（只傳新訂單進來）
async function fetchOrderDetails(entries) {
  const itemRe = /^(.+?)【(\d{1,2}\/\d{1,2})\s+(第\d+部)】(.+)$/

  function parseProductName(text) {
    const m = text.trim().match(itemRe)
    if (!m) return null
    return { member_name: m[1].trim(), raw_date: m[2], session: m[3], event_name: m[4].trim() }
  }

  function buildEventLabel(applyInfo, fallback) {
    if (applyInfo?.singleNum) {
      let s = `${applyInfo.singleNum}${applyInfo.singleSuffix}シングル`
      if (applyInfo.singleTitle)  s += `「${applyInfo.singleTitle}」`
      if (applyInfo.lotteryRound) s += `/${applyInfo.lotteryRound}`
      return s
    }
    if (applyInfo?.albumNum) {
      let s = `${applyInfo.albumNum}${applyInfo.albumSuffix}アルバム`
      if (applyInfo.albumTitle)   s += `「${applyInfo.albumTitle}」`
      if (applyInfo.lotteryRound) s += `/${applyInfo.lotteryRound}`
      return s
    }
    if (applyInfo?.isAlbum) {
      let s = 'アルバム'
      if (applyInfo.albumTitle)   s += `「${applyInfo.albumTitle}」`
      if (applyInfo.lotteryRound) s += `/${applyInfo.lotteryRound}`
      return s
    }
    return fallback
  }

  const parser  = new DOMParser()
  const records = []

  for (const { id, info: applyInfo } of entries) {
    let res
    try { res = await fetch(`/mypage/apply_detail/${id}/`) } catch { continue }
    if (!res.ok) continue

    const detailDoc   = parser.parseFromString(await res.text(), 'text/html')
    const sourceBase  = `https://fortunemusic.jp/mypage/apply_detail/${id}/`
    const aggregated  = {}

    detailDoc.querySelectorAll('tbody tr:not(.tblCatLast)').forEach(row => {
      const nameTd = row.querySelector('td:first-child')
      if (!nameTd) return
      const parsed = parseProductName(nameTd.textContent)
      if (!parsed) return

      const quaCells = row.querySelectorAll('td.tdQua')
      const applied  = parseInt((quaCells[0]?.textContent || '').match(/\d+/)?.[0] || '0')
      const won      = parseInt((quaCells[1]?.textContent || '').match(/\d+/)?.[0] || '0')
      const key      = parsed.member_name + parsed.raw_date + parsed.session

      if (aggregated[key]) {
        aggregated[key].applied_count += applied
        aggregated[key].won_count     += won
      } else {
        const eventMonth = parseInt(parsed.raw_date.split('/')[0])
        const eventYear  = applyInfo
          ? (eventMonth < applyInfo.month ? applyInfo.year + 1 : applyInfo.year)
          : new Date().getFullYear()

        const sourceURL    = `${sourceBase}#${encodeURIComponent(parsed.member_name)}|${parsed.raw_date}|${parsed.session}`
        const eventLabel   = buildEventLabel(applyInfo, parsed.event_name)
        const slashIdx     = eventLabel.lastIndexOf('/')
        const singleName   = slashIdx !== -1 ? eventLabel.slice(0, slashIdx) : eventLabel
        const lotteryRound = slashIdx !== -1 ? eventLabel.slice(slashIdx + 1) : (applyInfo?.lotteryRound || '')
        const singleNumber = applyInfo?.singleNum ? parseInt(applyInfo.singleNum) : 0

        aggregated[key] = {
          member_name:   parsed.member_name,
          event_date:    `${eventYear}/${parsed.raw_date}`,
          session:       parsed.session,
          single_number: singleNumber,
          single_name:   singleName,
          lottery_round: lotteryRound,
          applied_count: applied,
          won_count:     won,
          source_url:    sourceURL,
        }
      }
    })

    Object.values(aggregated).forEach(rec => records.push(rec))
  }

  return { records }
}
// ────────────────────────────────────────────────────────────────────────────

// 從 applyInfo 組出 single_name（不需要 detail page）
function buildSingleName(info) {
  if (!info) return null
  if (info.singleNum) {
    let s = `${info.singleNum}${info.singleSuffix}シングル`
    if (info.singleTitle) s += `「${info.singleTitle}」`
    return s
  }
  if (info.albumNum) {
    let s = `${info.albumNum}${info.albumSuffix}アルバム`
    if (info.albumTitle) s += `「${info.albumTitle}」`
    return s
  }
  if (info.isAlbum) {
    let s = 'アルバム'
    if (info.albumTitle) s += `「${info.albumTitle}」`
    return s
  }
  return null
}

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

// 步驟二：逐頁掃描 → 新訂單抓詳情 → 舊訂單更新 title
scrapeBtn.addEventListener('click', async () => {
  const { [BACKEND_KEY]: backendUrl, [TOKEN_KEY]: scrapeToken } =
    await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])

  scrapeBtn.disabled    = true
  scrapeBtn.textContent = '抓取中...'

  let totalNew     = 0
  let totalSkipped = 0
  let totalUpdated = 0
  let page         = 1

  try {
    const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/mypage/apply_list/*' })
    if (tabs.length === 0) {
      showResult('error', '找不到申請列表分頁，請先點「同步」開啟頁面，確認登入後再試')
      return
    }

    while (true) {
      // ── 1. 掃描列表頁（快，不進 detail）──
      showResult('info', `正在掃描第 ${page} 頁...`)
      const listResult = await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func:   scrapeListPage,
        args:   [page],
      })
      const listData = listResult[0]?.result
      if (!listData)        { showResult('error', '掃描失敗，無法取得結果'); break }
      if (listData.error)   { showResult('error', listData.error);           break }
      if (!listData.hasMore) break

      const { orders } = listData

      // ── 2. 詢問後端哪些訂單是新的 ──
      const checkRes = await fetch(`${backendUrl}/scrape/check-orders`, {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ scrape_token: scrapeToken, order_ids: orders.map(o => o.id) }),
      })
      const checkJson = await checkRes.json()
      if (!checkRes.ok) { showResult('error', checkJson.error || '查詢失敗'); break }

      const newSet      = new Set(checkJson.new_order_ids)
      const existingSet = new Set(checkJson.existing_order_ids)

      // ── 3. 只抓新訂單的 detail page ──
      const newEntries = orders.filter(o => newSet.has(o.id))
      if (newEntries.length > 0) {
        showResult('info', `第 ${page} 頁：${newEntries.length} 筆新訂單，抓取中...`)
        const detailResult = await chrome.scripting.executeScript({
          target: { tabId: tabs[0].id },
          func:   fetchOrderDetails,
          args:   [newEntries],
        })
        const detailData = detailResult[0]?.result
        if (detailData?.records?.length > 0) {
          const pushRes = await fetch(`${backendUrl}/scrape/push`, {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({ scrape_token: scrapeToken, records: detailData.records }),
          })
          const pushJson = await pushRes.json()
          if (!pushRes.ok) { showResult('error', pushJson.error || '上傳失敗'); break }
          totalNew     += pushJson.new_records ?? 0
          totalSkipped += pushJson.skipped     ?? 0
        }
      }

      // ── 4. 既有訂單：批次更新 title（如有變動）──
      const titleUpdates = orders
        .filter(o => existingSet.has(o.id))
        .map(o => {
          const singleName = buildSingleName(o.info)
          if (!singleName) return null
          return {
            order_id:      o.id,
            single_name:   singleName,
            single_number: o.info?.singleNum ? parseInt(o.info.singleNum) : 0,
          }
        })
        .filter(Boolean)

      if (titleUpdates.length > 0) {
        const updateRes = await fetch(`${backendUrl}/scrape/update-titles`, {
          method:  'POST',
          headers: { 'Content-Type': 'application/json' },
          body:    JSON.stringify({ scrape_token: scrapeToken, updates: titleUpdates }),
        })
        const updateJson = await updateRes.json()
        if (updateRes.ok) totalUpdated += updateJson.updated ?? 0
      }

      page++
    }

    // ── 最終結果 ──
    const parts = []
    if (totalNew > 0)     parts.push(`新增 ${totalNew} 筆`)
    if (totalSkipped > 0) parts.push(`跳過重複 ${totalSkipped} 筆`)
    if (totalUpdated > 0) parts.push(`更新 title ${totalUpdated} 筆`)

    showResult(
      parts.length ? 'success' : 'warning',
      parts.length
        ? `同步完成！共 ${page - 1} 頁，${parts.join('、')}`
        : `同步完成，共 ${page - 1} 頁，無新資料`
    )
    setWaitingMode(false)
  } catch (e) {
    showResult('error', '發生錯誤：' + e.message)
  } finally {
    scrapeBtn.disabled    = false
    scrapeBtn.textContent = '開始抓取'
  }
})

init()
