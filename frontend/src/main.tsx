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

function renderFatalError(title: string, details: string) {
  const root = document.getElementById('root')
  if (!root) return
  const container = document.createElement('div')
  container.style.cssText = 'padding:24px;color:#c00;font-family:monospace;white-space:pre-wrap'
  const heading = document.createElement('strong')
  heading.textContent = title
  const content = document.createElement('pre')
  content.textContent = details
  container.append(heading, content)
  root.replaceChildren(container)
}

function showUnhandledError(title: string, details: string) {
  const existing = document.getElementById('unhandled-error-banner')
  if (existing) return
  const banner = document.createElement('div')
  banner.id = 'unhandled-error-banner'
  banner.style.cssText = 'position:fixed;bottom:0;left:0;right:0;padding:12px 24px;background:#c00;color:#fff;font-family:monospace;font-size:12px;z-index:99999;max-height:120px;overflow:auto;white-space:pre-wrap;cursor:pointer'
  const heading = document.createElement('strong')
  heading.textContent = title + ' '
  const content = document.createElement('span')
  content.textContent = details
  banner.append(heading, content)
  banner.addEventListener('click', () => banner.remove())
  document.body.appendChild(banner)
  console.error(title, details)
}

window.addEventListener('error', (event) => {
  const details = `${event.message}\n${event.filename}:${event.lineno}:${event.colno}\n\n${event.error?.stack || ''}`
  renderFatalError('界面脚本错误', details)
})

window.addEventListener('unhandledrejection', (event) => {
  const reason = event.reason
  const details = typeof reason === 'object' && reason !== null
    ? `${reason.message || reason}\n\n${reason.stack || ''}`
    : String(reason)
  showUnhandledError('未处理的异步错误:', details)
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
  const details = `${err}\n\n${(err as Error)?.stack || ''}`
  renderFatalError('界面渲染错误', details)
}
