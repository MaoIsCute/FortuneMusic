const EXTENSION_ID = 'gdclpkfeiocedicokoenhconeoocigeh'

export function detectExtension() {
  return new Promise((resolve) => {
    if (!window.chrome?.runtime?.sendMessage) {
      resolve(false)
      return
    }
    chrome.runtime.sendMessage(EXTENSION_ID, { type: 'PING' }, (response) => {
      const err = chrome.runtime.lastError
      if (!err) {
        resolve(true)
      } else if (err.message.includes('Could not establish connection')) {
        resolve(false)
      } else {
        // "message port closed" 等其他錯誤 = 有安裝但沒回應
        resolve(true)
      }
    })
  })
}

// 取得已安裝擴充功能的版本號（PING 回應裡帶的 manifest version），偵測不到時回傳 null
export function getExtensionVersion() {
  return new Promise((resolve) => {
    if (!window.chrome?.runtime?.sendMessage) {
      resolve(null)
      return
    }
    chrome.runtime.sendMessage(EXTENSION_ID, { type: 'PING' }, (response) => {
      if (chrome.runtime.lastError) { resolve(null); return }
      resolve(response?.version || null)
    })
  })
}
