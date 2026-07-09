const BACKEND_KEY = 'backendUrl'
const TOKEN_KEY   = 'scrapeToken'

const setupPage     = document.getElementById('setupPage')
const mainPage      = document.getElementById('mainPage')
const statusEl      = document.getElementById('status')
const syncBtn       = document.getElementById('syncBtn')
const scrapeBtn     = document.getElementById('scrapeBtn')
const fullSyncBtn      = document.getElementById('fullSyncBtn')
const fullScrapeBtn    = document.getElementById('fullScrapeBtn')
const fullStartEl      = document.getElementById('fullStart')
const fullEndEl        = document.getElementById('fullEnd')
const fullGroupEl      = document.getElementById('fullGroup')
const purchaseSyncBtn   = document.getElementById('purchaseSyncBtn')
const purchaseScrapeBtn = document.getElementById('purchaseScrapeBtn')
const stopBtn           = document.getElementById('stopBtn')
const verifyBtn             = document.getElementById('verifyBtn')
const refetchBtn            = document.getElementById('refetchBtn')
const verifyPurchaseBtn     = document.getElementById('verifyPurchaseBtn')
const refetchPurchaseBtn    = document.getElementById('refetchPurchaseBtn')

let isStopping = false
let verifyMissingEntries         = [] // [{id, info}] 個握驗證後缺漏的訂單
let verifyPurchaseMissingEntries = [] // [{id, urlId, info}] 花費驗證後缺漏的記錄

function showStopBtn(show) {
  stopBtn.style.display  = show ? 'inline-block' : 'none'
  stopBtn.textContent    = '停止'
  stopBtn.disabled       = false
}

stopBtn.addEventListener('click', () => {
  isStopping         = true
  stopBtn.textContent = '停止中...'
  stopBtn.disabled    = true
})

// 網站對場次的顯示方式不一致，同一個場次有時候寫「第1部」、有時候只寫「1部」，
// 沒有「第」的話統一補上，避免同一場次因為文字不一致被統計成兩筆
function normalizeSession(s) {
  if (!s) return s
  return s.startsWith('第') ? s : '第' + s
}

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
    const html = await res.text()
    if (html.includes('アクセスが集中')) return { error: `第 ${pageNum} 頁：網站流量限制，請稍後再試` }
    doc = new DOMParser().parseFromString(html, 'text/html')
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
      const sm    = eventText.match(/(\d+)(st|nd|rd|th)(?:シングル|SG)/)
      const am    = eventText.match(/(\d+)(st|nd|rd|th)アルバム/)
      const titleM = eventText.match(/[『「](.+?)[』」]/)
      const rm    = eventText.match(/第(\d+)次/)
      if (rm) info.lotteryRound = parseInt(rm[1])
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
      if (/乃木坂46/.test(eventText))      info.group = 'nogizaka46'
      else if (/櫻坂46/.test(eventText))   info.group = 'sakurazaka46'
      else if (/日向坂46/.test(eventText)) info.group = 'hinatazaka46'
    }

    orders.push({ id, info: Object.keys(info).length ? info : null })
  })

  if (orders.length === 0) return { orders: [], hasMore: false }
  return { orders, hasMore: true }
}

// ─── 階段二：只抓新訂單的 detail page（4 個並行一批）──────────────────────────
// entries = [{id, info}]（只傳新訂單進來）
async function fetchOrderDetails(entries) {
  const CONCURRENCY = 1
  const itemRe = /^(.+?)【(\d{1,2}\/\d{1,2})[^】]*(第?\d+部)】(.+)$/
  function toHalf(s) {
    return (s || '').replace(/[！-～]/g, c => String.fromCharCode(c.charCodeAt(0) - 0xFEE0))
  }

  function parseProductName(text) {
    const m = toHalf(text.trim()).match(itemRe)
    if (!m) return null
    return { member_name: m[1].trim().replace(/[\s　]+/g, ''), raw_date: m[2], session: normalizeSession(m[3]), event_name: m[4].trim() }
  }

  function buildEventLabel(applyInfo, fallback) {
    if (applyInfo?.singleNum) {
      let s = `${applyInfo.singleNum}${applyInfo.singleSuffix}シングル`
      if (applyInfo.singleTitle)  s += `『${applyInfo.singleTitle}』`
      return s
    }
    if (applyInfo?.albumNum) {
      let s = `${applyInfo.albumNum}${applyInfo.albumSuffix}アルバム`
      if (applyInfo.albumTitle)   s += `『${applyInfo.albumTitle}』`
      return s
    }
    if (applyInfo?.isAlbum) {
      let s = 'アルバム'
      if (applyInfo.albumTitle)   s += `『${applyInfo.albumTitle}』`
      return s
    }
    return fallback
  }

  const parser = new DOMParser()

  async function fetchSingleOrder({ id, info: applyInfo }) {
    async function tryFetch() {
      let res
      try { res = await fetch(`/mypage/apply_detail/${id}/`) } catch { return null }
      if (!res.ok) return null
      const html = await res.text()
      if (html.includes('アクセスが集中')) return null
      return html
    }

    let html = await tryFetch()
    if (html === null) {
      await new Promise(r => setTimeout(r, 4000))
      html = await tryFetch()
    }
    if (html === null) return []

    const detailDoc  = parser.parseFromString(html, 'text/html')
    const sourceBase = `https://fortunemusic.jp/mypage/apply_detail/${id}/`
    const aggregated = {}

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
        const singleName   = buildEventLabel(applyInfo, parsed.event_name)
        const lotteryRound = applyInfo?.lotteryRound || 0
        const singleNumber = applyInfo?.singleNum ? parseInt(applyInfo.singleNum) : 0

        aggregated[key] = {
          group:         applyInfo?.group || '',
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

    return Object.values(aggregated)
  }

  const records = []
  for (let i = 0; i < entries.length; i += CONCURRENCY) {
    const batch = entries.slice(i, i + CONCURRENCY)
    const batchResults = await Promise.all(batch.map(fetchSingleOrder))
    batchResults.forEach(r => records.push(...r))
    if (i + CONCURRENCY < entries.length) await new Promise(r => setTimeout(r, 500))
  }

  return { records }
}
// ────────────────────────────────────────────────────────────────────────────

// 從 applyInfo 組出 single_name（不需要 detail page）
function buildSingleName(info) {
  if (!info) return null
  if (info.singleNum) {
    let s = `${info.singleNum}${info.singleSuffix}シングル`
    if (info.singleTitle) s += `『${info.singleTitle}』`
    return s
  }
  if (info.albumNum) {
    let s = `${info.albumNum}${info.albumSuffix}アルバム`
    if (info.albumTitle) s += `『${info.albumTitle}』`
    return s
  }
  if (info.isAlbum) {
    let s = 'アルバム'
    if (info.albumTitle) s += `『${info.albumTitle}』`
    return s
  }
  return null
}

async function init() {
  const { version } = chrome.runtime.getManifest()
  document.getElementById('versionBadge').textContent = `v${version}`
  applyGroupConstraints()
  const data = await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])
  if (data[BACKEND_KEY] && data[TOKEN_KEY]) showMain(data[BACKEND_KEY])
  else showSetup()
}

function showSetup() {
  setupPage.style.display = 'block'
  mainPage.style.display  = 'none'
}

function showMain(backendUrl) {
  setupPage.style.display = 'none'
  mainPage.style.display  = 'block'
  statusEl.textContent    = '已連接：' + backendUrl
  statusEl.className      = 'status connected'
}

const progressSection = document.getElementById('progressSection')
const progressLabel   = document.getElementById('progressLabel')
const progressBarFill = document.getElementById('progressBarFill')
const progressPct     = document.getElementById('progressPct')
const progressDetail  = document.getElementById('progressDetail')
const progressTimer   = document.getElementById('progressTimer')
const logList         = document.getElementById('logList')

let timerInterval = null
let timerStart    = null

function formatElapsed(seconds) {
  const m = String(Math.floor(seconds / 60)).padStart(2, '0')
  const s = String(seconds % 60).padStart(2, '0')
  return `${m}:${s}`
}

function startTimer() {
  timerStart = Date.now()
  progressTimer.style.display = 'block'
  progressTimer.textContent = '已執行 00:00'
  timerInterval = setInterval(() => {
    const elapsed = Math.floor((Date.now() - timerStart) / 1000)
    progressTimer.textContent = `已執行 ${formatElapsed(elapsed)}`
  }, 1000)
}

function stopTimer() {
  if (timerInterval) { clearInterval(timerInterval); timerInterval = null }
  progressTimer.style.display = 'none'
  return timerStart ? Math.floor((Date.now() - timerStart) / 1000) : 0
}

function updateProgress(label, current, total, detail) {
  progressSection.style.display = 'flex'
  progressLabel.textContent = '⏳ ' + label
  progressDetail.textContent = detail || ''
  if (total > 0) {
    const pct = Math.min(100, Math.round(current / total * 100))
    progressBarFill.classList.remove('indeterminate')
    progressBarFill.style.width = pct + '%'
    progressPct.textContent = pct + '%'
  } else {
    progressBarFill.classList.add('indeterminate')
    progressBarFill.style.width = ''
    progressPct.textContent = '-'
  }
}

function hideProgress() {
  progressSection.style.display = 'none'
  progressBarFill.classList.remove('indeterminate')
  progressBarFill.style.width = '0%'
}

function buildMismatchWarning(mismatches) {
  if (!mismatches || mismatches.length === 0) return null
  const ids = mismatches.slice(0, 5).map(m => m.id).join(', ') + (mismatches.length > 5 ? ' 等' : '')
  return `${mismatches.length} 筆訂單小計不符，可能漏抓（entry_id: ${ids}）`
}

function addLogEntry(type, newCount, skipCount, errorMsg, stopped = false, elapsed = null, warningMsg = null) {
  const empty = logList.querySelector('.log-empty')
  if (empty) empty.remove()

  const now = new Date()
  const t = String(now.getHours()).padStart(2, '0') + ':' + String(now.getMinutes()).padStart(2, '0')
  const isError   = !!errorMsg
  const isStopped = stopped && !isError
  const isEmpty   = !errorMsg && !stopped && newCount === 0 && skipCount === 0
  const hasWarning = !isError && !!warningMsg
  const cls  = isError ? 'error' : (isStopped || isEmpty || hasWarning) ? 'warning' : 'success'
  const icon = isError ? '❌' : (isStopped || isEmpty || hasWarning) ? '⚠️' : '✅'
  const timeStr = elapsed !== null ? ` · 耗時 ${formatElapsed(elapsed)}` : ''
  let body = isError   ? errorMsg
    : isStopped ? `已停止 · 新增 ${newCount} 筆${skipCount > 0 ? ' · 跳過 ' + skipCount : ''}${timeStr}`
    : isEmpty   ? `無新資料${timeStr}`
    : `新增 ${newCount} 筆${skipCount > 0 ? ' · 跳過 ' + skipCount : ''}${timeStr}`
  if (hasWarning) body += `<br>⚠️ ${warningMsg}`

  const el = document.createElement('div')
  el.className = 'log-entry ' + cls
  el.innerHTML =
    '<div class="log-entry-header"><span>' + icon + ' ' + type + '</span><span>' + t + '</span></div>' +
    '<div class="log-entry-body">' + body + '</div>'
  logList.insertBefore(el, logList.firstChild)
}

async function pushScrapeLog(backendUrl, scrapeToken, type, newCount, skipCount, error, durationSec = 0) {
  if (!backendUrl || !scrapeToken) return
  try {
    await fetch(backendUrl + '/scrape/log', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ scrape_token: scrapeToken, type, new_count: newCount, skip_count: skipCount, error: error || '', duration_sec: durationSec }),
    })
  } catch {}
}

function showResult(type, message) {
  progressSection.style.display = 'flex'
  progressBarFill.classList.remove('indeterminate')
  progressPct.textContent = ''
  progressDetail.textContent = message
  const cfg = {
    error:   { width: '100%', bg: '#dc2626', label: '❌ 錯誤' },
    warning: { width: '100%', bg: '#d97706', label: '⚠️ 發現缺漏' },
    success: { width: '100%', bg: '#059669', label: '✅ 完成' },
    info:    { width: '0%',   bg: '#7c3aed', label: 'ℹ️ 提示' },
  }[type] || { width: '0%', bg: '#7c3aed', label: 'ℹ️ 提示' }
  progressBarFill.style.width = cfg.width
  progressBarFill.style.background = cfg.bg
  progressLabel.textContent = cfg.label
}

function setWaitingMode(on) {
  syncBtn.style.display   = on ? 'none'  : 'block'
  scrapeBtn.style.display = on ? 'block' : 'none'
}

document.getElementById('openAppBtn').addEventListener('click', () => {
  chrome.tabs.create({ url: 'https://fortunemusic.vercel.app/scrape' })
})

document.getElementById('settingsBtn').addEventListener('click', showSetup)

async function waitForTabLoad(tabId, timeout = 8000) {
  return new Promise(resolve => {
    function onUpdated(id, info) {
      if (id === tabId && info.status === 'complete') {
        chrome.tabs.onUpdated.removeListener(onUpdated)
        resolve()
      }
    }
    chrome.tabs.onUpdated.addListener(onUpdated)
    setTimeout(() => { chrome.tabs.onUpdated.removeListener(onUpdated); resolve() }, timeout)
  })
}

// fortunemusic.jp tab 復用：有就導航，沒有才開新分頁
async function getOrOpenFortuneMusicTab(path) {
  const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/*' })
  if (tabs.length > 0) {
    await chrome.tabs.update(tabs[0].id, { url: `https://fortunemusic.jp${path}`, active: true })
    return tabs[0].id
  }
  const tab = await chrome.tabs.create({ url: `https://fortunemusic.jp${path}`, active: true })
  return tab.id
}

// 步驟一：開啟申請列表分頁（復用已開啟的 fortunemusic 分頁）
syncBtn.addEventListener('click', async () => {
  await getOrOpenFortuneMusicTab('/mypage/apply_list/')
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
  isStopping = false
  showStopBtn(true)
  startTimer()

  let totalNew     = 0
  let totalSkipped = 0
  let totalUpdated = 0
  let page         = 1
  let errorMsg     = ''
  let techError    = ''

  try {
    const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/mypage/apply_list/*' })
    if (tabs.length === 0) throw new Error('找不到申請列表分頁，請先點「同步」開啟頁面，確認登入後再試')

    while (true) {
      if (isStopping) break
      updateProgress('個握抽選', 0, 0, `掃描第 ${page} 頁...`)
      const listResult = await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func:   scrapeListPage,
        args:   [page],
      })
      const listData = listResult[0]?.result
      if (!listData)        { errorMsg = '掃描失敗，請確認申請列表頁面是否正常顯示'; techError = '掃描失敗：executeScript 回傳 null'; break }
      if (listData.error)   { errorMsg = techError = listData.error; break }
      if (!listData.hasMore) break

      const { orders } = listData

      const checkRes = await fetch(`${backendUrl}/scrape/check-orders`, {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ scrape_token: scrapeToken, order_ids: orders.map(o => o.id) }),
      })
      const checkJson = await checkRes.json()
      if (!checkRes.ok) { errorMsg = '查詢訂單時發生錯誤，請稍後再試'; techError = `check-orders 失敗：HTTP ${checkRes.status}，${checkJson.error || ''}`.trim(); break }

      const newSet      = new Set(checkJson.new_order_ids)
      const existingSet = new Set(checkJson.existing_order_ids)

      const newEntries = orders.filter(o => newSet.has(o.id))
      if (newEntries.length > 0) {
        updateProgress('個握抽選', 0, 0, `第 ${page} 頁：${newEntries.length} 筆新訂單，抓取中...`)
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
          if (!pushRes.ok) { errorMsg = '上傳紀錄時發生錯誤，請稍後再試'; techError = `push 失敗：HTTP ${pushRes.status}，${pushJson.error || ''}`.trim(); break }
          totalNew     += pushJson.new_records ?? 0
          totalSkipped += pushJson.skipped     ?? 0
        }
      }

      const titleUpdates = orders
        .filter(o => existingSet.has(o.id))
        .map(o => {
          const singleName = buildSingleName(o.info)
          if (!singleName) return null
          return {
            order_id:      o.id,
            group:         o.info?.group || '',
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

    const elapsed = stopTimer(); hideProgress()
    addLogEntry('個握抽選', totalNew, totalSkipped, errorMsg || null, isStopping, elapsed)
    await pushScrapeLog(backendUrl, scrapeToken, '個握抽選', totalNew, totalSkipped, techError || errorMsg, elapsed)
    if (!errorMsg && !isStopping) setWaitingMode(false)
  } catch (e) {
    const elapsed = stopTimer(); hideProgress()
    addLogEntry('個握抽選', totalNew, totalSkipped, '抓取過程發生未預期錯誤，請稍後再試', false, elapsed)
    await pushScrapeLog(backendUrl, scrapeToken, '個握抽選', totalNew, totalSkipped, e.message, elapsed)
  } finally {
    scrapeBtn.disabled    = false
    scrapeBtn.textContent = '開始抓取'
    showStopBtn(false)
  }
})

// ─── 全握：直接打 ticket-api.fortunemeets.app ───────────────────────────────

function ordinalSuffix(n) {
  const v = n % 100, d = n % 10
  if (v >= 11 && v <= 13) return 'th'
  return d === 1 ? 'st' : d === 2 ? 'nd' : d === 3 ? 'rd' : 'th'
}

function parseLotteryRound(times) {
  const m = (times || '').match(/(\d+)次/)
  if (!m) return 1.0
  const n = parseFloat(m[1])
  return times.includes('保障') ? n + 0.5 : n
}

function parseFullApiResults(results, singleNum, group) {
  const suffix = ordinalSuffix(singleNum)
  const singleName = `${singleNum}${suffix}シングル`
  const recordMap = {}

  for (const item of results) {
    const prizeInfo = item.prizeInfo || {}
    const dateStr = prizeInfo.date || ''
    const eventName = prizeInfo.event || ''
    const atIdx = dateStr.indexOf('＠')
    const eventType = eventName.includes('リアル') ? '実体' : '線上'
    const venue = atIdx >= 0 ? dateStr.slice(atIdx + 1).trim() : ''

    const dm = dateStr.match(/(\d{4})年(\d{1,2})月(\d{1,2})日/)
    if (!dm) continue
    const eventDate = `${dm[1]}/${parseInt(dm[2])}/${parseInt(dm[3])}`

    const session = prizeInfo.part || ''
    const memberName = (prizeInfo.members || [])
      .map(m => m.replace(/[\s　]+/g, '').trim()).filter(Boolean).join('・')
    if (!memberName) continue

    const appliedCount = item.count || 0
    const wonCount = parseInt(item.resultInfo?.win || '0')
    const lotteryRound = parseLotteryRound(prizeInfo.times || '')

    const orderId = `full:${group}_${singleNum}${suffix}:${item.prizeId}:${memberName}:${lotteryRound}`

    if (recordMap[orderId]) {
      recordMap[orderId].applied_count += appliedCount
      recordMap[orderId].won_count     += wonCount
    } else {
      recordMap[orderId] = {
        order_id:      orderId,
        group,
        single_number: singleNum,
        single_name:   singleName,
        event_type:    eventType,
        venue,
        event_date:    eventDate,
        session,
        member_name:   memberName,
        applied_count: appliedCount,
        won_count:     wonCount,
        lottery_round: lotteryRound,
        source_url:    '',
      }
    }
  }
  return Object.values(recordMap)
}
// ─────────────────────────────────────────────────────────────────────────────

const FORTUNE_USER_ID_KEY = 'fortuneUserId'

const GROUP_MIN_SINGLE  = { nogizaka46: 25, hinatazaka46: 4, sakurazaka46: 4 }
const GROUP_DEFAULT_END = { nogizaka46: 42, hinatazaka46: 17, sakurazaka46: 15 }
const fullGroupHintEl   = document.getElementById('fullGroupHint')

function applyGroupConstraints() {
  const group = fullGroupEl.value
  const min   = GROUP_MIN_SINGLE[group] ?? 1
  fullStartEl.min = min
  fullEndEl.min   = min
  if (min > 1) {
    fullStartEl.value = min
    fullEndEl.value    = GROUP_DEFAULT_END[group] ?? min
    const groupLabel = fullGroupEl.options[fullGroupEl.selectedIndex]?.text || group
    fullGroupHintEl.innerHTML = `ℹ️ ${groupLabel} 最早收錄於系統的資料為 ${min}單。<br>起始單最小為 ${min}。`
    fullGroupHintEl.style.display = 'block'
  } else {
    fullGroupHintEl.style.display = 'none'
  }
}

fullGroupEl.addEventListener('change', applyGroupConstraints)

function setFullWaitingMode(on) {
  fullSyncBtn.style.display   = on ? 'none'  : 'block'
  fullScrapeBtn.style.display = on ? 'block' : 'none'
}

// 步驟一：開啟 ticket.fortunemeets.app，讀取並儲存 lscache-id
fullSyncBtn.addEventListener('click', async () => {
  const group   = fullGroupEl.value || 'nogizaka46'
  const openNum = parseInt(fullStartEl.value) || parseInt(fullEndEl.value) || 1
  const openUrl = `https://ticket.fortunemeets.app/${group}/${openNum}${ordinalSuffix(openNum)}#/history`

  const tabs = await chrome.tabs.query({ url: 'https://ticket.fortunemeets.app/*' })
  let tabId
  if (tabs.length > 0) {
    tabId = tabs[0].id
    await chrome.tabs.update(tabId, { url: openUrl, active: true })
  } else {
    const tab = await chrome.tabs.create({ url: openUrl, active: true })
    tabId = tab.id
    await new Promise(resolve => {
      function onUpdated(id, info) {
        if (id === tabId && info.status === 'complete') {
          chrome.tabs.onUpdated.removeListener(onUpdated)
          resolve()
        }
      }
      chrome.tabs.onUpdated.addListener(onUpdated)
      setTimeout(() => { chrome.tabs.onUpdated.removeListener(onUpdated); resolve() }, 8000)
    })
  }

  const result = await chrome.scripting.executeScript({
    target: { tabId },
    func: () => {
      const dump = (store) => {
        const out = {}
        for (let i = 0; i < store.length; i++) {
          const k = store.key(i)
          out[k] = store.getItem(k)
        }
        return out
      }
      return {
        local:   dump(localStorage),
        session: dump(sessionStorage),
        cookie:  document.cookie,
      }
    },
  })
  const { local, session, cookie } = result[0]?.result ?? {}

  const raw = local?.['lscache-id'] ?? session?.['lscache-id'] ?? null
  const userId = raw ? raw.replace(/"/g, '') : null

  if (!userId) {
    const fmt = (obj) => Object.entries(obj ?? {}).map(([k, v]) => `${k}: ${String(v).slice(0, 80)}`).join('\n') || '（空）'
    showResult('error',
      `無法取得使用者 ID。\n\nlocalStorage:\n${fmt(local)}\n\nsessionStorage:\n${fmt(session)}\n\ncookie:\n${(cookie || '（空）').slice(0, 400)}`)
    return
  }

  await chrome.storage.local.set({ [FORTUNE_USER_ID_KEY]: userId })
  setFullWaitingMode(true)
  showResult('info',
    '已連線 ticket.fortunemeets.app。\n\n' +
    '① 選擇團體與單曲範圍\n' +
    '② 點「開始全握抓取」'
  )
})

// 步驟二：透過 ticket.fortunemeets.app tab 執行 fetch（使用瀏覽器 session）
fullScrapeBtn.addEventListener('click', async () => {
  const stored = await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY, FORTUNE_USER_ID_KEY])
  const backendUrl = stored[BACKEND_KEY]
  const scrapeToken = stored[TOKEN_KEY]
  const userId = stored[FORTUNE_USER_ID_KEY]

  if (!userId) {
    showResult('error', '尚未連線，請先點「全握同步」')
    return
  }

  const ticketTabs = await chrome.tabs.query({ url: 'https://ticket.fortunemeets.app/*' })
  if (ticketTabs.length === 0) {
    showResult('error', '請先點「全握同步」開啟 ticket.fortunemeets.app')
    return
  }
  const tabId = ticketTabs[0].id

  const group    = fullGroupEl.value || 'nogizaka46'
  const startNum = parseInt(fullStartEl.value) || 1
  const endNum   = parseInt(fullEndEl.value)   || 0

  if (endNum > 0 && endNum < startNum) {
    showResult('error', `結束單（${endNum}）不能小於起始單（${startNum}）`)
    return
  }

  fullScrapeBtn.disabled    = true
  fullScrapeBtn.textContent = '抓取中...'
  isStopping = false
  showStopBtn(true)
  startTimer()

  let totalNew = 0, totalSkipped = 0, emptyStreak = 0
  const MAX_EMPTY    = 3
  const totalSingles = endNum > 0 ? endNum - startNum + 1 : 0
  let errorMsg = '', techError = ''

  try {
    for (let n = startNum; ; n++) {
      if (endNum > 0 && n > endNum) break
      if (isStopping) break

      const suffix      = ordinalSuffix(n)
      const artistEvent = `${group}_${n}${suffix}`

      updateProgress('全握', n - startNum, totalSingles,
        `正在抓取 ${artistEvent}... 新增 ${totalNew} · 跳過 ${totalSkipped}`)

      let apiResult
      try {
        const execResult = await chrome.scripting.executeScript({
          target: { tabId },
          func: async (uid, event) => {
            try {
              const res = await fetch('https://ticket-api.fortunemeets.app/user/history2', {
                headers: { 'x-user-id': uid, 'x-artist-event': event },
              })
              const body = await res.text()
              return { status: res.status, body }
            } catch (e) {
              return { status: 0, body: e.message }
            }
          },
          args: [userId, artistEvent],
        })
        apiResult = execResult[0]?.result
      } catch (e) {
        errorMsg = '全握腳本執行失敗，請重新整理頁面後再試'
        techError = `腳本執行錯誤：${e.message}`
        break
      }

      if (!apiResult || apiResult.status === 0) {
        errorMsg = '無法連線至全握服務，請確認網路狀態'
        techError = `網路錯誤：${apiResult?.body ?? '未知'}`
        break
      }
      if (apiResult.status === 401) {
        await chrome.storage.local.remove(FORTUNE_USER_ID_KEY)
        errorMsg = techError = '登入已過期，請重新點「全握同步」'
        break
      }
      if (apiResult.status === 500) {
        let parsed = null
        try { parsed = JSON.parse(apiResult.body) } catch {}
        if (parsed?.error === 'InternalFailureException') {
          emptyStreak++
          if (endNum === 0 && emptyStreak >= MAX_EMPTY) break
          continue
        }
        errorMsg = `全握服務暫時無法使用（${artistEvent}），請稍後再試`
        techError = `API 錯誤：500（${artistEvent}）\n${apiResult.body.slice(0, 200)}`
        break
      }
      if (apiResult.status !== 200) {
        errorMsg = `全握服務回應異常（${artistEvent}），請稍後再試`
        techError = `API 錯誤：${apiResult.status}（${artistEvent}）\n${apiResult.body.slice(0, 200)}`
        break
      }

      let data
      try { data = JSON.parse(apiResult.body) } catch { errorMsg = '全握資料格式異常，請稍後再試'; techError = 'JSON 解析失敗'; break }
      if (data.error) { errorMsg = '全握 API 回傳錯誤，請稍後再試'; techError = data.message || 'API 錯誤'; break }

      const results = data.results || []
      if (results.length === 0) {
        emptyStreak++
        if (endNum === 0 && emptyStreak >= MAX_EMPTY) break
        continue
      }
      emptyStreak = 0

      const records = parseFullApiResults(results, n, group)
      if (records.length > 0) {
        const pushRes = await fetch(`${backendUrl}/scrape/full/push`, {
          method:  'POST',
          headers: { 'Content-Type': 'application/json' },
          body:    JSON.stringify({ scrape_token: scrapeToken, records }),
        })
        const pushJson = await pushRes.json()
        if (!pushRes.ok) { errorMsg = '上傳紀錄時發生錯誤，請稍後再試'; techError = `full/push 失敗：HTTP ${pushRes.status}，${pushJson.error || ''}`.trim(); break }
        totalNew     += pushJson.new_records ?? 0
        totalSkipped += pushJson.skipped     ?? 0
      }
    }

    const elapsed = stopTimer(); hideProgress()
    addLogEntry('全握', totalNew, totalSkipped, errorMsg || null, isStopping, elapsed)
    await pushScrapeLog(backendUrl, scrapeToken, '全握', totalNew, totalSkipped, techError || errorMsg, elapsed)
    if (!errorMsg && !isStopping) setFullWaitingMode(false)
  } catch (e) {
    const elapsed = stopTimer(); hideProgress()
    addLogEntry('全握', totalNew, totalSkipped, '抓取過程發生未預期錯誤，請稍後再試', false, elapsed)
    await pushScrapeLog(backendUrl, scrapeToken, '全握', totalNew, totalSkipped, e.message, elapsed)
  } finally {
    fullScrapeBtn.disabled    = false
    fullScrapeBtn.textContent = '開始全握抓取'
    showStopBtn(false)
  }
})

// ─── 購入記錄：掃描 entry_list ───────────────────────────────────────────────
async function scrapeEntryListPage() {
  function toHalf(s) {
    return (s || '').replace(/[！-～]/g, c => String.fromCharCode(c.charCodeAt(0) - 0xFEE0))
  }

  const doc = document
  const _url   = window.location.href
  const _title = document.title
  const _body  = document.body?.innerText?.slice(0, 300) || ''

  if (doc.querySelector('[name="login_form"], input[type="password"]'))
    return { error: '頁面顯示登入表單，請先登入後再點「開始抓取」', _url, _title, _body }

  const entries = []
  const seen = {}

  doc.querySelectorAll('a[href]').forEach(a => {
    const m = (a.getAttribute('href') || '').match(/\/mypage\/entry_detail\/(\d+)\/?/)
    if (!m) return
    const urlId       = m[1]            // URL 數字 ID（用於 fetch detail page）
    const orderNumber = a.textContent.trim()  // 訂單號碼（唯一識別，dedup 用）
    if (!orderNumber || seen[orderNumber]) return
    seen[orderNumber] = true
    const row = a.closest('tr')
    if (!row) { entries.push({ id: orderNumber, urlId, info: { orderNumber } }); return }

    let appliedAt = '', description = ''
    row.querySelectorAll('td').forEach(td => {
      const hdg = td.querySelector('span.hdg')
      if (hdg?.textContent.trim() === '申込日時') {
        appliedAt = td.textContent.replace('申込日時', '').trim()
      }
      if (!td.querySelector('span.hdg') && !td.querySelector('a[href*="entry_detail"]')) {
        const t = td.textContent.trim()
        if (t) description = toHalf(t)
      }
    })

    // イベント名 td 有 span.hdg，直接用整行 textContent 做全部比對
    const rowText = toHalf(row.textContent || '')

    const info = { orderNumber, appliedAt, description }
    const sm = rowText.match(/(\d+)(st|nd|rd|th)(?:シングル|SG)/)
    const am = rowText.match(/(\d+)(st|nd|rd|th)アルバム/)
    const titleM = rowText.match(/[『「](.+?)[』」]/)
    const rm = rowText.match(/第(\d+)次/)
    if (rm) info.lotteryRound = parseInt(rm[1])
    if (sm) {
      info.singleNum = sm[1]; info.singleSuffix = sm[2]
      if (titleM) info.singleTitle = titleM[1]
    } else if (am || /アルバム/.test(rowText)) {
      info.isAlbum = true
      if (am) { info.albumNum = am[1]; info.albumSuffix = am[2] }
      if (titleM) info.albumTitle = titleM[1]
    }
    if (/乃木坂46/.test(rowText))      info.group = 'nogizaka46'
    else if (/櫻坂46/.test(rowText))   info.group = 'sakurazaka46'
    else if (/日向坂46/.test(rowText)) info.group = 'hinatazaka46'

    entries.push({ id: orderNumber, urlId, info })
  })

  const nextLink = Array.from(doc.querySelectorAll('a[href*="/mypage/entry_list/?page="]'))
    .find(a => a.textContent.trim() === '次へ')
  const nextUrl  = nextLink ? nextLink.getAttribute('href') : null

  if (entries.length === 0) return { entries: [], hasMore: false, nextUrl, _url, _title, _body }
  return { entries, hasMore: true, nextUrl, _url, _title, _body }
}

// entry_detail から明細を取得（4 個並行）
async function fetchEntryDetailItems(entries) {
  const CONCURRENCY = 1
  const itemRe = /^(.+?)【(\d{1,2}\/\d{1,2})[^】]*(第?\d+部)】/
  const parser = new DOMParser()
  function toHalf(s) {
    return (s || '').replace(/[！-～]/g, c => String.fromCharCode(c.charCodeAt(0) - 0xFEE0))
  }

  function buildSingleName(info) {
    if (!info) return ''
    if (info.singleNum) {
      let s = `${info.singleNum}${info.singleSuffix}シングル`
      if (info.singleTitle) s += `『${info.singleTitle}』`
      return s
    }
    if (info.isAlbum) {
      let s = info.albumNum ? `${info.albumNum}${info.albumSuffix}アルバム` : 'アルバム'
      if (info.albumTitle) s += `『${info.albumTitle}』`
      return s
    }
    // 不符合シングル/アルバム規則時（如 DVD/寫真集發售紀念等個握商品），
    // fallback 回傳 entry_list 抓到的原始商品名稱，避免變成空字串
    return info.description || ''
  }

  async function fetchSingle({ id, urlId, info }) {
    const url = `/mypage/entry_detail/${urlId || id}/`
    async function tryFetch() {
      let res
      try { res = await fetch(url) } catch { return null }
      if (res.status === 403 || res.status === 503) return null
      if (!res.ok) return null
      return await res.text()
    }
    let html = await tryFetch()
    if (html === null) {
      await new Promise(r => setTimeout(r, 15000))
      html = await tryFetch()
    }
    if (html === null) {
      await new Promise(r => setTimeout(r, 15000))
      html = await tryFetch()
    }
    if (html === null) return { items: [], mismatch: null }
    const doc = parser.parseFromString(html, 'text/html')
    const aggregated = {}

    let appliedYear = new Date().getFullYear()
    let appliedMonth = new Date().getMonth() + 1
    if (info?.appliedAt) {
      const dm = info.appliedAt.match(/(\d{4})-(\d{1,2})-/)
      if (dm) { appliedYear = parseInt(dm[1]); appliedMonth = parseInt(dm[2]) }
    }

    doc.querySelectorAll('tbody tr').forEach(row => {
      const firstTd = row.querySelector('td:first-child')
      if (!firstTd) return
      const text = toHalf(firstTd.textContent.trim())
      const m = text.match(itemRe)
      if (!m) return

      const memberName = m[1].trim().replace(/[\s　]+/g, '')
      const rawDate    = m[2]
      const session    = normalizeSession(m[3])

      // 単価
      let unitPrice = 0
      row.querySelectorAll('td.tdAmo').forEach(td => {
        if (td.querySelector('span.hdg')) {
          const n = td.textContent.replace(/[^0-9]/g, '')
          if (n) unitPrice = parseInt(n)
        }
      })

      // 数量
      const quaTd  = row.querySelector('td.tdQua')
      const qtyStr = (quaTd?.textContent || '').replace(/[^0-9]/g, '')
      const quantity = parseInt(qtyStr) || 1
      const subtotal = unitPrice * quantity

      // 同張訂單內相同 member+date+session 視為同一筆，加總 quantity/subtotal 後只送一筆
      // （避免後端 item_key 去重時把第二筆當成「已存在」而漏算金額）
      const key = memberName + rawDate + session
      if (aggregated[key]) {
        aggregated[key].quantity += quantity
        aggregated[key].subtotal += subtotal
        return
      }

      const eventMonth = parseInt(rawDate.split('/')[0])
      const eventYear  = eventMonth < appliedMonth ? appliedYear + 1 : appliedYear

      aggregated[key] = {
        entry_id:      id,
        group:         info?.group || '',
        order_number:  info?.orderNumber || '',
        member_name:   memberName,
        event_date:    `${eventYear}/${rawDate}`,
        session,
        single_number: info?.singleNum ? parseInt(info.singleNum) : 0,
        single_name:   buildSingleName(info),
        lottery_round: info?.lotteryRound || 0,
        unit_price:    unitPrice,
        quantity,
        subtotal,
        applied_at:    info?.appliedAt || '',
      }
    })

    const items = Object.values(aggregated)

    // 用頁面上顯示的「小計」跟解析出的 items 加總比對，抓出漏抓/誤抓的訂單
    let declaredSubtotal = null
    doc.querySelectorAll('span.hdg').forEach(span => {
      if (span.textContent.trim() !== '小計') return
      const n = (span.parentElement?.textContent || '').replace(/[^0-9]/g, '')
      if (n) declaredSubtotal = parseInt(n)
    })
    const actualSubtotal = items.reduce((sum, it) => sum + it.subtotal, 0)
    const mismatch = (declaredSubtotal !== null && declaredSubtotal !== actualSubtotal)
      ? { id, declared: declaredSubtotal, actual: actualSubtotal }
      : null

    return { items, mismatch }
  }

  const purchases = []
  const mismatches = []
  for (let i = 0; i < entries.length; i += CONCURRENCY) {
    const batch = entries.slice(i, i + CONCURRENCY)
    const results = await Promise.all(batch.map(fetchSingle))
    results.forEach(r => {
      purchases.push(...r.items)
      if (r.mismatch) mismatches.push(r.mismatch)
    })
    if (i + CONCURRENCY < entries.length) await new Promise(r => setTimeout(r, 2000))
  }
  return { purchases, mismatches }
}
// ─────────────────────────────────────────────────────────────────────────────

function setPurchaseWaitingMode(on) {
  purchaseSyncBtn.style.display   = on ? 'none'  : 'block'
  purchaseScrapeBtn.style.display = on ? 'block' : 'none'
}

// 步驟一：開啟購入記錄分頁（復用已開啟的 fortunemusic 分頁）
purchaseSyncBtn.addEventListener('click', async () => {
  await getOrOpenFortuneMusicTab('/mypage/entry_list/')
  setPurchaseWaitingMode(true)
  showResult('info',
    '已開啟購入記錄頁面。\n\n' +
    '① 如未登入，請先登入\n' +
    '② 確認可看到購入記錄後\n' +
    '③ 點「開始抓取」'
  )
})

// 步驟二：先掃完所有列表頁收集 entry，再集中抓 detail（避免 detail fetch 觸發 rate limit 導致列表翻頁 403）
purchaseScrapeBtn.addEventListener('click', async () => {
  const { [BACKEND_KEY]: backendUrl, [TOKEN_KEY]: scrapeToken } =
    await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])

  purchaseScrapeBtn.disabled    = true
  purchaseScrapeBtn.textContent = '抓取中...'
  isStopping = false
  showStopBtn(true)
  startTimer()

  let totalNew = 0, totalSkipped = 0, page = 1, errorMsg = '', techError = ''
  const allMismatches = []

  try {
    const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/mypage/entry_list/*' })
    if (tabs.length === 0) throw new Error('找不到購入記錄分頁，請先點「同步」開啟頁面，確認登入後再試')

    // 階段一：掃描所有列表頁（不做任何 detail fetch）
    const allEntries = []
    while (true) {
      if (isStopping) break
      updateProgress('個握花費', 0, 0, `掃描列表第 ${page} 頁...（已收集 ${allEntries.length} 筆）`)
      const listResult = await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func:   scrapeEntryListPage,
        args:   [],
      })
      const listData = listResult[0]?.result
      if (!listData)      { errorMsg = '掃描失敗，請確認購入記錄頁面是否正常顯示'; techError = '掃描失敗：executeScript 回傳 null'; break }
      if (listData.error) { errorMsg = listData.error; techError = `${listData.error} | URL:${listData._url} | title:${listData._title} | body:${listData._body}`; break }
      if (!listData.hasMore) break

      allEntries.push(...listData.entries)
      if (!listData.nextUrl) break

      await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func: (href) => {
          const link = document.querySelector(`a[href="${href}"]`)
            || Array.from(document.querySelectorAll('a[href*="/mypage/entry_list/?page="]'))
               .find(a => a.textContent.trim() === '次へ')
          if (link) link.click()
        },
        args: [listData.nextUrl],
      })
      await waitForTabLoad(tabs[0].id)
      await new Promise(r => setTimeout(r, 1000))
      page++
    }

    // 階段二：查重
    if (!errorMsg && !isStopping && allEntries.length > 0) {
      updateProgress('個握花費', 0, 0, `比對 ${allEntries.length} 筆...`)
      const checkRes = await fetch(`${backendUrl}/scrape/check-entries`, {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ scrape_token: scrapeToken, entry_ids: allEntries.map(e => e.id) }),
      })
      const checkJson = await checkRes.json()
      if (!checkRes.ok) {
        errorMsg = '查詢花費記錄時發生錯誤，請稍後再試'
        techError = `check-entries 失敗：HTTP ${checkRes.status}，${checkJson.error || ''}`.trim()
      } else {
        const newSet     = new Set(checkJson.new_entry_ids)
        const newEntries = allEntries.filter(e => newSet.has(e.id))
        totalSkipped     = allEntries.length - newEntries.length

        // 階段三：抓取新訂單明細
        if (newEntries.length > 0) {
          updateProgress('個握花費', 0, 0, `抓取 ${newEntries.length} 筆新記錄明細...`)
          const detailResult = await chrome.scripting.executeScript({
            target: { tabId: tabs[0].id },
            func:   fetchEntryDetailItems,
            args:   [newEntries],
          })
          const detailData = detailResult[0]?.result
          if (detailData?.mismatches?.length > 0) allMismatches.push(...detailData.mismatches)
          if (detailData?.purchases?.length > 0) {
            const pushRes = await fetch(`${backendUrl}/scrape/purchases/push`, {
              method:  'POST',
              headers: { 'Content-Type': 'application/json' },
              body:    JSON.stringify({ scrape_token: scrapeToken, purchases: detailData.purchases }),
            })
            const pushJson = await pushRes.json()
            if (!pushRes.ok) {
              errorMsg = '上傳花費記錄時發生錯誤，請稍後再試'
              techError = `purchases/push 失敗：HTTP ${pushRes.status}，${pushJson.error || ''}`.trim()
            } else {
              totalNew      = pushJson.new_purchases ?? 0
              totalSkipped += pushJson.skipped       ?? 0
            }
          }
        }
      }
    }

    const elapsed = stopTimer(); hideProgress()
    const warningMsg = buildMismatchWarning(allMismatches)
    addLogEntry('個握花費', totalNew, totalSkipped, errorMsg || null, isStopping, elapsed, warningMsg)
    await pushScrapeLog(backendUrl, scrapeToken, '個握花費', totalNew, totalSkipped, techError || errorMsg, elapsed)
    chrome.tabs.update(tabs[0].id, { url: 'https://fortunemusic.jp/mypage/entry_list/' })
    if (!errorMsg && !isStopping) setPurchaseWaitingMode(false)
  } catch (e) {
    const elapsed = stopTimer(); hideProgress()
    addLogEntry('個握花費', totalNew, totalSkipped, '抓取過程發生未預期錯誤，請稍後再試', false, elapsed, buildMismatchWarning(allMismatches))
    await pushScrapeLog(backendUrl, scrapeToken, '個握花費', totalNew, totalSkipped, e.message, elapsed)
  } finally {
    purchaseScrapeBtn.disabled    = false
    purchaseScrapeBtn.textContent = '開始抓取'
    showStopBtn(false)
  }
})

// ─── 驗證完整性 ──────────────────────────────────────────────────────────────
verifyBtn.addEventListener('click', async () => {
  const { [BACKEND_KEY]: backendUrl, [TOKEN_KEY]: scrapeToken } =
    await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])

  const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/mypage/apply_list/*' })
  if (tabs.length === 0) {
    await getOrOpenFortuneMusicTab('/mypage/apply_list/')
    showResult('info', '已開啟申請列表頁面。\n確認已登入後，再點「驗證完整性」')
    return
  }

  verifyBtn.disabled = true
  refetchBtn.style.display = 'none'
  verifyMissingEntries = []
  isStopping = false
  showStopBtn(true)

  let page = 1
  const allOrders = []
  let errorMsg = ''

  try {
    while (true) {
      if (isStopping) break
      updateProgress('驗證完整性', 0, 0, `掃描第 ${page} 頁...（已找到 ${allOrders.length} 筆）`)

      const listResult = await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func:   scrapeListPage,
        args:   [page],
      })
      const listData = listResult[0]?.result
      if (!listData)       { errorMsg = '掃描失敗，無法取得結果'; break }
      if (listData.error)  { errorMsg = listData.error;           break }
      if (!listData.hasMore) break

      allOrders.push(...listData.orders)
      page++
    }

    if (!errorMsg && !isStopping) {
      updateProgress('驗證完整性', 0, 0, `比對 ${allOrders.length} 筆訂單...`)

      const checkRes = await fetch(`${backendUrl}/scrape/check-orders`, {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ scrape_token: scrapeToken, order_ids: allOrders.map(o => o.id) }),
      })
      const checkJson = await checkRes.json()

      if (!checkRes.ok) {
        errorMsg = checkJson.error || '比對失敗'
      } else {
        const newSet = new Set(checkJson.new_order_ids || [])
        verifyMissingEntries = allOrders.filter(o => newSet.has(o.id))
        const missing = verifyMissingEntries.length
        const total   = allOrders.length

        hideProgress()
        if (missing === 0) {
          showResult('success', `網站共 ${total} 筆訂單，全數已在 DB 中 ✓`)
        } else {
          showResult('warning', `網站共 ${total} 筆 · DB 缺少 ${missing} 筆`)
          refetchBtn.textContent  = `補抓遺漏 (${missing} 筆)`
          refetchBtn.style.display = 'block'
        }
      }
    }

    if (isStopping) {
      hideProgress()
      showResult('info', `已停止，掃描到第 ${page} 頁（共 ${allOrders.length} 筆）`)
    }
    if (errorMsg) {
      hideProgress()
      showResult('error', errorMsg)
    }
  } catch (e) {
    hideProgress()
    showResult('error', e.message)
  } finally {
    verifyBtn.disabled = false
    showStopBtn(false)
  }
})

// ─── 補抓遺漏 ────────────────────────────────────────────────────────────────
refetchBtn.addEventListener('click', async () => {
  if (verifyMissingEntries.length === 0) return

  const { [BACKEND_KEY]: backendUrl, [TOKEN_KEY]: scrapeToken } =
    await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])

  const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/*' })
  if (tabs.length === 0) {
    showResult('error', '找不到 fortunemusic.jp 分頁，請先點「驗證完整性」開啟頁面')
    return
  }

  refetchBtn.disabled = true
  isStopping = false
  showStopBtn(true)
  startTimer()

  const entries    = [...verifyMissingEntries]
  const BATCH      = 4
  let totalNew = 0, totalSkipped = 0, errorMsg = '', techError = ''

  try {
    for (let i = 0; i < entries.length; i += BATCH) {
      if (isStopping) break
      const batch = entries.slice(i, i + BATCH)

      updateProgress('補抓遺漏', i, entries.length,
        `${i + 1}～${Math.min(i + BATCH, entries.length)} / ${entries.length} 筆`)

      const detailResult = await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func:   fetchOrderDetails,
        args:   [batch],
      })
      const detailData = detailResult[0]?.result
      if (detailData?.records?.length > 0) {
        const pushRes = await fetch(`${backendUrl}/scrape/push`, {
          method:  'POST',
          headers: { 'Content-Type': 'application/json' },
          body:    JSON.stringify({ scrape_token: scrapeToken, records: detailData.records }),
        })
        const pushJson = await pushRes.json()
        if (!pushRes.ok) { errorMsg = '上傳紀錄時發生錯誤，請稍後再試'; techError = `push 失敗：HTTP ${pushRes.status}，${pushJson.error || ''}`.trim(); break }
        totalNew     += pushJson.new_records ?? 0
        totalSkipped += pushJson.skipped     ?? 0
      }
    }

    const elapsed = stopTimer(); hideProgress()
    addLogEntry('補抓遺漏', totalNew, totalSkipped, errorMsg || null, isStopping, elapsed)
    await pushScrapeLog(backendUrl, scrapeToken, '補抓遺漏', totalNew, totalSkipped, techError || errorMsg, elapsed)

    if (!errorMsg && !isStopping) {
      verifyMissingEntries    = []
      refetchBtn.style.display = 'none'
    }
  } catch (e) {
    const elapsed = stopTimer(); hideProgress()
    addLogEntry('補抓遺漏', totalNew, totalSkipped, '補抓過程發生未預期錯誤，請稍後再試', false, elapsed)
    await pushScrapeLog(backendUrl, scrapeToken, '補抓遺漏', totalNew, totalSkipped, e.message, elapsed)
  } finally {
    refetchBtn.disabled     = false
    refetchBtn.textContent  = `補抓遺漏 (${verifyMissingEntries.length} 筆)`
    showStopBtn(false)
  }
})

// ─── 花費記錄：驗證完整性 ────────────────────────────────────────────────────
verifyPurchaseBtn.addEventListener('click', async () => {
  const { [BACKEND_KEY]: backendUrl, [TOKEN_KEY]: scrapeToken } =
    await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])

  const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/mypage/entry_list/*' })
  if (tabs.length === 0) {
    await getOrOpenFortuneMusicTab('/mypage/entry_list/')
    showResult('info', '已開啟購入記錄頁面。\n確認已登入後，再點「驗證完整性」')
    return
  }

  verifyPurchaseBtn.disabled = true
  refetchPurchaseBtn.style.display = 'none'
  verifyPurchaseMissingEntries = []
  isStopping = false
  showStopBtn(true)
  startTimer()

  let page = 1
  const allEntries = []
  let errorMsg = ''

  try {
    while (true) {
      if (isStopping) break
      updateProgress('驗證個握花費完整性', 0, 0, `掃描第 ${page} 頁...（已找到 ${allEntries.length} 筆）`)

      const listResult = await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func:   scrapeEntryListPage,
        args:   [],
      })
      const listData = listResult[0]?.result
      if (!listData)       { errorMsg = '掃描失敗，無法取得結果'; break }
      if (listData.error)  { errorMsg = listData.error;           break }
      if (!listData.hasMore) break

      allEntries.push(...listData.entries)

      if (!listData.nextUrl) break

      await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func: (href) => {
          const link = document.querySelector(`a[href="${href}"]`)
            || Array.from(document.querySelectorAll('a[href*="/mypage/entry_list/?page="]'))
               .find(a => a.textContent.trim() === '次へ')
          if (link) link.click()
        },
        args: [listData.nextUrl],
      })
      await waitForTabLoad(tabs[0].id)
      await new Promise(r => setTimeout(r, 1000))

      page++
    }

    if (!errorMsg && !isStopping) {
      updateProgress('驗證個握花費完整性', 0, 0, `比對 ${allEntries.length} 筆記錄...`)

      const checkRes = await fetch(`${backendUrl}/scrape/check-entries`, {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ scrape_token: scrapeToken, entry_ids: allEntries.map(e => e.id) }),
      })
      const checkJson = await checkRes.json()

      if (!checkRes.ok) {
        errorMsg = checkJson.error || '比對失敗'
      } else {
        const newSet = new Set(checkJson.new_entry_ids || [])
        verifyPurchaseMissingEntries = allEntries.filter(e => newSet.has(e.id))
        const missing = verifyPurchaseMissingEntries.length
        const total   = allEntries.length

        stopTimer(); hideProgress()
        if (missing === 0) {
          showResult('success', `網站共 ${total} 筆購入記錄，全數已在 DB 中 ✓`)
        } else {
          showResult('warning', `網站共 ${total} 筆 · DB 缺少 ${missing} 筆`)
          refetchPurchaseBtn.textContent   = `補抓遺漏 (${missing} 筆)`
          refetchPurchaseBtn.style.display = 'block'
        }
        return
      }
    }

    if (isStopping) { stopTimer(); hideProgress(); showResult('info', `已停止，掃描到第 ${page} 頁（共 ${allEntries.length} 筆）`) }
    if (errorMsg)   { stopTimer(); hideProgress(); showResult('error', errorMsg) }
  } catch (e) {
    stopTimer(); hideProgress()
    showResult('error', e.message)
  } finally {
    verifyPurchaseBtn.disabled = false
    showStopBtn(false)
  }
})

// ─── 花費記錄：補抓遺漏 ──────────────────────────────────────────────────────
refetchPurchaseBtn.addEventListener('click', async () => {
  if (verifyPurchaseMissingEntries.length === 0) return

  const { [BACKEND_KEY]: backendUrl, [TOKEN_KEY]: scrapeToken } =
    await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])

  const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/*' })
  if (tabs.length === 0) {
    showResult('error', '找不到 fortunemusic.jp 分頁，請先點「驗證完整性」開啟頁面')
    return
  }

  refetchPurchaseBtn.disabled = true
  isStopping = false
  showStopBtn(true)
  startTimer()

  const entries = [...verifyPurchaseMissingEntries]
  let totalNew = 0, totalSkipped = 0, errorMsg = '', techError = ''
  const allMismatches = []

  try {
    for (let i = 0; i < entries.length; i++) {
      if (isStopping) break

      updateProgress('補抓個握花費遺漏', i, entries.length,
        `${i + 1} / ${entries.length} 筆`)

      const detailResult = await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func:   fetchEntryDetailItems,
        args:   [[entries[i]]],
      })
      const detailData = detailResult[0]?.result
      if (detailData?.mismatches?.length > 0) allMismatches.push(...detailData.mismatches)
      if (detailData?.purchases?.length > 0) {
        const pushRes = await fetch(`${backendUrl}/scrape/purchases/push`, {
          method:  'POST',
          headers: { 'Content-Type': 'application/json' },
          body:    JSON.stringify({ scrape_token: scrapeToken, purchases: detailData.purchases }),
        })
        const pushJson = await pushRes.json()
        if (!pushRes.ok) { errorMsg = '上傳花費記錄時發生錯誤，請稍後再試'; techError = `purchases/push 失敗：HTTP ${pushRes.status}，${pushJson.error || ''}`.trim(); break }
        totalNew     += pushJson.new_purchases ?? 0
        totalSkipped += pushJson.skipped       ?? 0
      }

      if (i + 1 < entries.length) await new Promise(r => setTimeout(r, 500))
    }

    const elapsed = stopTimer(); hideProgress()
    addLogEntry('補抓個握花費遺漏', totalNew, totalSkipped, errorMsg || null, isStopping, elapsed, buildMismatchWarning(allMismatches))
    await pushScrapeLog(backendUrl, scrapeToken, '補抓個握花費遺漏', totalNew, totalSkipped, techError || errorMsg)

    if (!errorMsg && !isStopping) {
      verifyPurchaseMissingEntries     = []
      refetchPurchaseBtn.style.display = 'none'
    }
  } catch (e) {
    const elapsed = stopTimer(); hideProgress()
    addLogEntry('補抓個握花費遺漏', totalNew, totalSkipped, '補抓過程發生未預期錯誤，請稍後再試', false, elapsed, buildMismatchWarning(allMismatches))
    await pushScrapeLog(backendUrl, scrapeToken, '補抓個握花費遺漏', totalNew, totalSkipped, e.message)
  } finally {
    refetchPurchaseBtn.disabled    = false
    refetchPurchaseBtn.textContent = `補抓遺漏 (${verifyPurchaseMissingEntries.length} 筆)`
    showStopBtn(false)
  }
})

// 授權成功後自動切換到主頁面（不需要重新開啟 popup）
chrome.storage.onChanged.addListener((changes, area) => {
  if (area !== 'local') return
  if (changes.scrapeToken || changes.backendUrl) {
    chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY], (data) => {
      if (data[BACKEND_KEY] && data[TOKEN_KEY]) showMain(data[BACKEND_KEY])
    })
  }
})

init()
