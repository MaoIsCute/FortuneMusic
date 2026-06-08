const EXTENSION_ID = 'gdclpkfeiocedicokoenhconeoocigeh'

export function detectExtension() {
  return new Promise((resolve) => {
    if (!window.chrome?.runtime?.sendMessage) {
      resolve(false)
      return
    }
    chrome.runtime.sendMessage(EXTENSION_ID, { type: 'PING' }, (response) => {
      resolve(!chrome.runtime.lastError && response?.pong === true)
    })
  })
}
