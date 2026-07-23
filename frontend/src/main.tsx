import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import { MultiProfileDock } from './components/MultiProfileDock'
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

window.addEventListener('error', (event) => {
  const el = document.getElementById('root')
  if (el) el.innerHTML = `<div style="padding:24px;color:#c00;font-family:monospace;white-space:pre-wrap"><b>JS Error</b>\n${event.message}\n${event.filename}:${event.lineno}:${event.colno}\n\n${event.error?.stack || ''}</div>`
})
window.addEventListener('unhandledrejection', (event) => {
  const el = document.getElementById('root')
  if (el) el.innerHTML = `<div style="padding:24px;color:#c00;font-family:monospace;white-space:pre-wrap"><b>Unhandled Promise Rejection</b>\n${event.reason}\n\n${event.reason?.stack || ''}</div>`
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
  window.addEventListener('keydown', (e) => {
    if (e.ctrlKey && e.shiftKey && e.key === 'D') {
      const d = document.getElementById('wails-diag')
      if (d) d.style.display = d.style.display === 'none' ? 'block' : 'none'
    }
  })
  createRoot(root).render(<StrictMode><App /><MultiProfileDock /></StrictMode>)
} catch (err) {
  const el = document.getElementById('root')
  if (el) el.innerHTML = `<div style="padding:24px;color:#c00;font-family:monospace;white-space:pre-wrap"><b>Render Error</b>\n${err}\n\n${(err as Error)?.stack || ''}</div>`
}
