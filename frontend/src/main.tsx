import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import './styles.css'
import './kernel.css'
import './runtime.css'
import './credential.css'
import './diagnostics.css'
import './evidence.css'
import './adapter.css'
import './lifecycle.css'
import './localRecovery.css'
import './multiProfile.css'
import './bulkLifecycle.css'
import './product.css'

window.addEventListener('error', (event) => {
  const el = document.getElementById('root')
  if (el) el.innerHTML = `<div style="padding:24px;color:#c00;font-family:monospace;white-space:pre-wrap"><b>界面脚本错误</b>\n${event.message}\n${event.filename}:${event.lineno}:${event.colno}\n\n${event.error?.stack || ''}</div>`
})

window.addEventListener('unhandledrejection', (event) => {
  const el = document.getElementById('root')
  if (el) el.innerHTML = `<div style="padding:24px;color:#c00;font-family:monospace;white-space:pre-wrap"><b>未处理的异步错误</b>\n${event.reason}\n\n${event.reason?.stack || ''}</div>`
})

const diagnostics: string[] = []
try {
  const hasGo = Boolean(window.go)
  diagnostics.push(`window.go: ${hasGo}`)
  if (hasGo) {
    diagnostics.push(`window.go.main: ${Boolean((window.go as any)?.main)}`)
    diagnostics.push(`DesktopApp: ${Boolean((window.go as any)?.main?.DesktopApp)}`)
    const methods = Object.keys((window.go as any)?.main?.DesktopApp || {})
    diagnostics.push(`methods: ${methods.length} [${methods.slice(0, 5).join(', ')}…]`)
  }
  diagnostics.push(`location: ${window.location?.href}`)
} catch (err) {
  diagnostics.push(`diagnostic error: ${err}`)
}

try {
  const root = document.getElementById('root')!
  const diagDiv = document.createElement('div')
  diagDiv.id = 'wails-diag'
  diagDiv.style.cssText = 'display:none;padding:8px;font-family:monospace;font-size:11px;background:#ff0;color:#000;white-space:pre-wrap'
  diagDiv.textContent = diagnostics.join('\n')
  root.appendChild(diagDiv)
  window.addEventListener('keydown', (event) => {
    if (event.ctrlKey && event.shiftKey && event.key === 'D') {
      const diagnostic = document.getElementById('wails-diag')
      if (diagnostic) diagnostic.style.display = diagnostic.style.display === 'none' ? 'block' : 'none'
    }
  })
  createRoot(root).render(<StrictMode><App /></StrictMode>)
} catch (err) {
  const el = document.getElementById('root')
  if (el) el.innerHTML = `<div style="padding:24px;color:#c00;font-family:monospace;white-space:pre-wrap"><b>界面渲染错误</b>\n${err}\n\n${(err as Error)?.stack || ''}</div>`
}
