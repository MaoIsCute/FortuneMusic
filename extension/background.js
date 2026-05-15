// 點擴充功能圖示時開啟 Side Panel
chrome.sidePanel
  .setPanelBehavior({ openPanelOnActionClick: true })
  .catch(() => {})
