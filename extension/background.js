// 點擴充功能圖示時開啟 Side Panel
chrome.sidePanel
  .setPanelBehavior({ openPanelOnActionClick: true })
  .catch(() => {})

// 接受來自 web app 的訊息
chrome.runtime.onMessageExternal.addListener((message, sender, sendResponse) => {
  if (message.type === 'PING') {
    sendResponse({ pong: true, version: chrome.runtime.getManifest().version })
    return
  }
  if (message.type !== 'FORTUNE_SETUP') return
  const { token, backendUrl } = message
  if (!token || !backendUrl) {
    sendResponse({ success: false })
    return
  }
  chrome.storage.local.set({ scrapeToken: token, backendUrl }, () => {
    sendResponse({ success: true })
  })
  return true // 保持通道開啟以等待非同步 sendResponse
})
