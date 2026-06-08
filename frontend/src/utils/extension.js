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
