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

let isStopping = false

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
    }

    orders.push({ id, info: Object.keys(info).length ? info : null })
  })

  if (orders.length === 0) return { orders: [], hasMore: false }
  return { orders, hasMore: true }
}

// ─── 階段二：只抓新訂單的 detail page（4 個並行一批）──────────────────────────
// entries = [{id, info}]（只傳新訂單進來）
async function fetchOrderDetails(entries) {
  const CONCURRENCY = 4
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
      return s
    }
    if (applyInfo?.albumNum) {
      let s = `${applyInfo.albumNum}${applyInfo.albumSuffix}アルバム`
      if (applyInfo.albumTitle)   s += `「${applyInfo.albumTitle}」`
      return s
    }
    if (applyInfo?.isAlbum) {
      let s = 'アルバム'
      if (applyInfo.albumTitle)   s += `「${applyInfo.albumTitle}」`
      return s
    }
    return fallback
  }

  const parser = new DOMParser()

  async function fetchSingleOrder({ id, info: applyInfo }) {
    let res
    try { res = await fetch(`/mypage/apply_detail/${id}/`) } catch { return [] }
    if (!res.ok) return []

    const detailDoc  = parser.parseFromString(await res.text(), 'text/html')
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
const logList         = document.getElementById('logList')

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

function addLogEntry(type, newCount, skipCount, errorMsg, stopped = false) {
  const empty = logList.querySelector('.log-empty')
  if (empty) empty.remove()

  const now = new Date()
  const t = String(now.getHours()).padStart(2, '0') + ':' + String(now.getMinutes()).padStart(2, '0')
  const isError   = !!errorMsg
  const isStopped = stopped && !isError
  const isEmpty   = !errorMsg && !stopped && newCount === 0 && skipCount === 0
  const cls  = isError ? 'error' : (isStopped || isEmpty) ? 'warning' : 'success'
  const icon = isError ? '❌' : (isStopped || isEmpty) ? '⚠️' : '✅'
  const body = isError   ? errorMsg
    : isStopped ? `已停止 · 新增 ${newCount} 筆${skipCount > 0 ? ' · 跳過 ' + skipCount : ''}`
    : isEmpty   ? '無新資料'
    : `新增 ${newCount} 筆${skipCount > 0 ? ' · 跳過 ' + skipCount : ''}`

  const el = document.createElement('div')
  el.className = 'log-entry ' + cls
  el.innerHTML =
    '<div class="log-entry-header"><span>' + icon + ' ' + type + '</span><span>' + t + '</span></div>' +
    '<div class="log-entry-body">' + body + '</div>'
  logList.insertBefore(el, logList.firstChild)
}

async function pushScrapeLog(backendUrl, scrapeToken, type, newCount, skipCount, error) {
  if (!backendUrl || !scrapeToken) return
  try {
    await fetch(backendUrl + '/scrape/log', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ scrape_token: scrapeToken, type, new_count: newCount, skip_count: skipCount, error: error || '' }),
    })
  } catch {}
}

function showResult(type, message) {
  // 保留給 sync 按鈕的提示訊息（顯示在 progressSection）
  progressSection.style.display = 'flex'
  progressBarFill.classList.remove('indeterminate')
  progressBarFill.style.width = type === 'error' ? '100%' : '0%'
  progressBarFill.style.background = type === 'error' ? '#dc2626' : '#7c3aed'
  progressPct.textContent = ''
  progressLabel.textContent = type === 'error' ? '❌ 錯誤' : type === 'info' ? 'ℹ️ 提示' : '✅ 完成'
  progressDetail.textContent = message
}

function setWaitingMode(on) {
  syncBtn.style.display   = on ? 'none'  : 'block'
  scrapeBtn.style.display = on ? 'block' : 'none'
}

document.getElementById('openAppBtn').addEventListener('click', () => {
  chrome.tabs.create({ url: 'https://fortunemusic.vercel.app/scrape' })
})

document.getElementById('settingsBtn').addEventListener('click', showSetup)

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

  let totalNew     = 0
  let totalSkipped = 0
  let totalUpdated = 0
  let page         = 1
  let errorMsg     = ''

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
      if (!listData)        { errorMsg = '掃描失敗，無法取得結果'; break }
      if (listData.error)   { errorMsg = listData.error;           break }
      if (!listData.hasMore) break

      const { orders } = listData

      const checkRes = await fetch(`${backendUrl}/scrape/check-orders`, {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ scrape_token: scrapeToken, order_ids: orders.map(o => o.id) }),
      })
      const checkJson = await checkRes.json()
      if (!checkRes.ok) { errorMsg = checkJson.error || '查詢失敗'; break }

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
          if (!pushRes.ok) { errorMsg = pushJson.error || '上傳失敗'; break }
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

    hideProgress()
    addLogEntry('個握抽選', totalNew, totalSkipped, errorMsg || null, isStopping)
    await pushScrapeLog(backendUrl, scrapeToken, '個握抽選', totalNew, totalSkipped, errorMsg)
    if (!errorMsg && !isStopping) setWaitingMode(false)
  } catch (e) {
    hideProgress()
    addLogEntry('個握抽選', totalNew, totalSkipped, e.message)
    await pushScrapeLog(backendUrl, scrapeToken, '個握抽選', totalNew, totalSkipped, e.message)
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
  const records = []
  const seen = new Set()

  for (const item of results) {
    const prizeInfo = item.prizeInfo || {}
    const dateStr = prizeInfo.date || ''
    const atIdx = dateStr.indexOf('＠')
    const eventType = atIdx >= 0 ? '実体' : '線上'
    const venue = atIdx >= 0 ? dateStr.slice(atIdx + 1).trim() : ''

    const dm = dateStr.match(/(\d{4})年(\d{1,2})月(\d{1,2})日/)
    if (!dm) continue
    const eventDate = `${dm[1]}/${parseInt(dm[2])}/${parseInt(dm[3])}`

    const session = prizeInfo.part || ''
    const memberName = ((prizeInfo.members || [])[0] || '').replace(/　/g, '')
    if (!memberName || !session) continue

    const appliedCount = item.count || 0
    const wonCount = parseInt(item.resultInfo?.win || '0')
    const lotteryRound = parseLotteryRound(prizeInfo.times || '')

    const orderId = `full:${group}_${singleNum}${suffix}:${item.prizeId}:${memberName}:${lotteryRound}`
    if (seen.has(orderId)) continue
    seen.add(orderId)

    records.push({
      order_id:      orderId,
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
    })
  }
  return records
}
// ─────────────────────────────────────────────────────────────────────────────

const FORTUNE_USER_ID_KEY = 'fortuneUserId'

function setFullWaitingMode(on) {
  fullSyncBtn.style.display   = on ? 'none'  : 'block'
  fullScrapeBtn.style.display = on ? 'block' : 'none'
}

// 步驟一：開啟 ticket.fortunemeets.app，讀取並儲存 lscache-id
fullSyncBtn.addEventListener('click', async () => {
  const tabs = await chrome.tabs.query({ url: 'https://ticket.fortunemeets.app/*' })
  let tabId
  if (tabs.length > 0) {
    tabId = tabs[0].id
    await chrome.tabs.update(tabId, { active: true })
  } else {
    const tab = await chrome.tabs.create({ url: 'https://ticket.fortunemeets.app/', active: true })
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
      const raw = localStorage.getItem('lscache-id')
      return raw ? raw.replace(/"/g, '') : null
    },
  })
  const userId = result[0]?.result

  if (!userId) {
    showResult('error', '無法取得使用者 ID，請確認已登入 ticket.fortunemeets.app')
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

// 步驟二：直接打 API 抓取，不需要切換分頁
fullScrapeBtn.addEventListener('click', async () => {
  const stored = await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY, FORTUNE_USER_ID_KEY])
  const backendUrl = stored[BACKEND_KEY]
  const scrapeToken = stored[TOKEN_KEY]
  const userId = stored[FORTUNE_USER_ID_KEY]

  if (!userId) {
    showResult('error', '尚未連線，請先點「全握同步」')
    return
  }

  const group    = fullGroupEl.value || 'nogizaka46'
  const startNum = parseInt(fullStartEl.value) || 1
  const endNum   = parseInt(fullEndEl.value)   || 0

  fullScrapeBtn.disabled    = true
  fullScrapeBtn.textContent = '抓取中...'
  isStopping = false
  showStopBtn(true)

  let totalNew = 0, totalSkipped = 0, emptyStreak = 0
  const MAX_EMPTY    = 3
  const totalSingles = endNum > 0 ? endNum - startNum + 1 : 0
  let errorMsg = ''

  try {
    for (let n = startNum; ; n++) {
      if (endNum > 0 && n > endNum) break
      if (isStopping) break

      const suffix      = ordinalSuffix(n)
      const artistEvent = `${group}_${n}${suffix}`

      updateProgress('全握', n - startNum, totalSingles,
        `正在抓取 ${artistEvent}... 新增 ${totalNew} · 跳過 ${totalSkipped}`)

      let apiRes
      try {
        apiRes = await fetch('https://ticket-api.fortunemeets.app/user/history2', {
          headers: { 'x-user-id': userId, 'x-artist-event': artistEvent },
        })
      } catch (e) {
        errorMsg = `網路錯誤：${e.message}`
        break
      }

      if (!apiRes.ok) {
        if (apiRes.status === 401) {
          await chrome.storage.local.remove(FORTUNE_USER_ID_KEY)
          errorMsg = '登入已過期，請重新點「全握同步」'
        } else {
          errorMsg = `API 錯誤：${apiRes.status}`
        }
        break
      }

      const data = await apiRes.json()
      if (data.error) { errorMsg = data.message || 'API 錯誤'; break }

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
        if (!pushRes.ok) { errorMsg = pushJson.error || '上傳失敗'; break }
        totalNew     += pushJson.new_records ?? 0
        totalSkipped += pushJson.skipped     ?? 0
      }
    }

    hideProgress()
    addLogEntry('全握', totalNew, totalSkipped, errorMsg || null, isStopping)
    await pushScrapeLog(backendUrl, scrapeToken, '全握', totalNew, totalSkipped, errorMsg)
    if (!errorMsg && !isStopping) setFullWaitingMode(false)
  } catch (e) {
    hideProgress()
    addLogEntry('全握', totalNew, totalSkipped, e.message)
    await pushScrapeLog(backendUrl, scrapeToken, '全握', totalNew, totalSkipped, e.message)
  } finally {
    fullScrapeBtn.disabled    = false
    fullScrapeBtn.textContent = '開始全握抓取'
    showStopBtn(false)
  }
})

// ─── 購入記錄：掃描 entry_list ───────────────────────────────────────────────
async function scrapeEntryListPage(pageNum) {
  function toHalf(s) {
    return (s || '').replace(/[！-～]/g, c => String.fromCharCode(c.charCodeAt(0) - 0xFEE0))
  }

  let doc
  if (pageNum === 1) {
    doc = document
    if (doc.querySelector('[name="login_form"], input[type="password"]'))
      return { error: '頁面顯示登入表單，請先登入後再點「開始抓取」' }
  } else {
    let res
    try { res = await fetch(`/mypage/entry_list/?page=${pageNum - 1}`) } catch (e) {
      return { error: `第 ${pageNum} 頁讀取失敗：${e.message}` }
    }
    if (!res.ok) return { error: `第 ${pageNum} 頁回應錯誤：${res.status}` }
    doc = new DOMParser().parseFromString(await res.text(), 'text/html')
  }

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
    const sm = rowText.match(/(\d+)(st|nd|rd|th)シングル/)
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

    entries.push({ id: orderNumber, urlId, info })
  })

  if (entries.length === 0) return { entries: [], hasMore: false }
  return { entries, hasMore: true }
}

// entry_detail から明細を取得（4 個並行）
async function fetchEntryDetailItems(entries) {
  const CONCURRENCY = 1
  const itemRe = /^(.+?)【(\d{1,2}\/\d{1,2})\s+(第\d+部)】/
  const parser = new DOMParser()

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
    return ''
  }

  async function fetchSingle({ id, urlId, info }) {
    let res
    try { res = await fetch(`/mypage/entry_detail/${urlId || id}/`) } catch { return [] }
    if (res.status === 503) {
      await new Promise(r => setTimeout(r, 2000))
      try { res = await fetch(`/mypage/entry_detail/${urlId || id}/`) } catch { return [] }
    }
    if (!res.ok) return []

    const doc = parser.parseFromString(await res.text(), 'text/html')
    const items = []

    let appliedYear = new Date().getFullYear()
    let appliedMonth = new Date().getMonth() + 1
    if (info?.appliedAt) {
      const dm = info.appliedAt.match(/(\d{4})-(\d{1,2})-/)
      if (dm) { appliedYear = parseInt(dm[1]); appliedMonth = parseInt(dm[2]) }
    }

    doc.querySelectorAll('tbody tr').forEach(row => {
      const firstTd = row.querySelector('td:first-child')
      if (!firstTd) return
      const text = firstTd.textContent.trim()
      const m = text.match(itemRe)
      if (!m) return

      const memberName = m[1].trim()
      const rawDate    = m[2]
      const session    = m[3]

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

      const eventMonth = parseInt(rawDate.split('/')[0])
      const eventYear  = eventMonth < appliedMonth ? appliedYear + 1 : appliedYear

      items.push({
        entry_id:      id,
        order_number:  info?.orderNumber || '',
        member_name:   memberName,
        event_date:    `${eventYear}/${rawDate}`,
        session,
        single_number: info?.singleNum ? parseInt(info.singleNum) : 0,
        single_name:   buildSingleName(info),
        lottery_round: info?.lotteryRound || 0,
        unit_price:    unitPrice,
        quantity,
        subtotal:      unitPrice * quantity,
        applied_at:    info?.appliedAt || '',
      })
    })

    return items
  }

  const purchases = []
  for (let i = 0; i < entries.length; i += CONCURRENCY) {
    const batch = entries.slice(i, i + CONCURRENCY)
    const results = await Promise.all(batch.map(fetchSingle))
    results.forEach(r => purchases.push(...r))
    if (i + CONCURRENCY < entries.length) await new Promise(r => setTimeout(r, 500))
  }
  return { purchases }
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

// 步驟二：逐頁掃描 entry_list → 新訂單抓 entry_detail
purchaseScrapeBtn.addEventListener('click', async () => {
  const { [BACKEND_KEY]: backendUrl, [TOKEN_KEY]: scrapeToken } =
    await chrome.storage.local.get([BACKEND_KEY, TOKEN_KEY])

  purchaseScrapeBtn.disabled    = true
  purchaseScrapeBtn.textContent = '抓取中...'
  isStopping = false
  showStopBtn(true)

  let totalNew = 0, totalSkipped = 0, page = 1, errorMsg = ''

  try {
    const tabs = await chrome.tabs.query({ url: 'https://fortunemusic.jp/mypage/entry_list/*' })
    if (tabs.length === 0) throw new Error('找不到購入記錄分頁，請先點「同步」開啟頁面，確認登入後再試')

    while (true) {
      if (isStopping) break
      updateProgress('個握花費', 0, 0, `掃描第 ${page} 頁...　新增 ${totalNew} · 跳過 ${totalSkipped}`)
      const listResult = await chrome.scripting.executeScript({
        target: { tabId: tabs[0].id },
        func:   scrapeEntryListPage,
        args:   [page],
      })
      const listData = listResult[0]?.result
      if (!listData)        { errorMsg = '掃描失敗，無法取得結果'; break }
      if (listData.error)   { errorMsg = listData.error;           break }
      if (!listData.hasMore) break

      const { entries } = listData

      const checkRes = await fetch(`${backendUrl}/scrape/check-entries`, {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ scrape_token: scrapeToken, entry_ids: entries.map(e => e.id) }),
      })
      const checkJson = await checkRes.json()
      if (!checkRes.ok) { errorMsg = checkJson.error || '查詢失敗'; break }

      const newSet     = new Set(checkJson.new_entry_ids)
      const newEntries = entries.filter(e => newSet.has(e.id))

      if (newEntries.length > 0) {
        updateProgress('個握花費', 0, 0, `第 ${page} 頁：${newEntries.length} 筆新記錄，抓取明細中...`)
        const detailResult = await chrome.scripting.executeScript({
          target: { tabId: tabs[0].id },
          func:   fetchEntryDetailItems,
          args:   [newEntries],
        })
        const detailData = detailResult[0]?.result
        if (detailData?.purchases?.length > 0) {
          const pushRes = await fetch(`${backendUrl}/scrape/purchases/push`, {
            method:  'POST',
            headers: { 'Content-Type': 'application/json' },
            body:    JSON.stringify({ scrape_token: scrapeToken, purchases: detailData.purchases }),
          })
          const pushJson = await pushRes.json()
          if (!pushRes.ok) { errorMsg = pushJson.error || '上傳失敗'; break }
          totalNew     += pushJson.new_purchases ?? 0
          totalSkipped += pushJson.skipped       ?? 0
        }
      }

      page++
    }

    hideProgress()
    addLogEntry('個握花費', totalNew, totalSkipped, errorMsg || null, isStopping)
    await pushScrapeLog(backendUrl, scrapeToken, '個握花費', totalNew, totalSkipped, errorMsg)
    if (!errorMsg && !isStopping) setPurchaseWaitingMode(false)
  } catch (e) {
    hideProgress()
    addLogEntry('個握花費', totalNew, totalSkipped, e.message)
    await pushScrapeLog(backendUrl, scrapeToken, '個握花費', totalNew, totalSkipped, e.message)
  } finally {
    purchaseScrapeBtn.disabled    = false
    purchaseScrapeBtn.textContent = '開始抓取'
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
