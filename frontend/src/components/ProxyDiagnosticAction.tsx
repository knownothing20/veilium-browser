import { useState } from 'react'
import { formatDateTime } from '../i18n/format'
import { backend } from '../lib/backend'
import type { Profile, ProxyDiagnosticReport } from '../types'

export function ProxyDiagnosticAction({ profile, nativeMode }: { profile: Profile; nativeMode: boolean }) {
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [report, setReport] = useState<ProxyDiagnosticReport>()

  async function run() {
    setOpen(true)
    setBusy(true)
    setError('')
    try { setReport(await backend.runProxyDiagnostics(profile.id)) }
    catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { setBusy(false) }
  }

  return <>
    <button className="button compact secondary" title={nativeMode ? '检测代理路由' : '需要桌面运行时'} disabled={!nativeMode || busy} onClick={() => void run()}>{busy ? '检测中…' : '代理检测'}</button>
    {open && <div className="overlay diagnostic-overlay" onMouseDown={() => setOpen(false)}>
      <aside className="diagnostic-drawer" onMouseDown={(event) => event.stopPropagation()}>
        <header className="editor-header">
          <div><span className="eyebrow">实时网络测量</span><h2>代理检测</h2><p>{profile.name}</p></div>
          <button className="close-button" onClick={() => setOpen(false)} aria-label="关闭">×</button>
        </header>
        <div className="diagnostic-content">
          {busy && <div className="loading-block">正在检测当前选择的网络路由…</div>}
          {error && <div className="form-error">{error}</div>}
          {report && !busy && <>
            <div className="diagnostic-summary">
              <span className={`diagnostic-status ${report.status}`}>{diagnosticStatusLabel(report.status)}</span>
              <div><span>出口 IP</span><strong>{report.exitIp || '不可用'}</strong></div>
              <div><span>首字节</span><strong>{formatLatency(report.firstByteLatencyMs)}</strong></div>
              <div><span>总耗时</span><strong>{formatLatency(report.totalLatencyMs)}</strong></div>
            </div>
            <dl className="diagnostic-route">
              <div><dt>配置路由</dt><dd>{report.proxyDisplay}</dd></div>
              <div><dt>路由类型</dt><dd>{report.routeKind}</dd></div>
              <div><dt>临时代理桥</dt><dd>{report.bridgeKind || '不需要'}</dd></div>
              <div><dt>完成时间</dt><dd>{formatDateTime(report.completedAt)}</dd></div>
            </dl>
            <div className="diagnostic-checks">
              {(report.checks || []).map((check) => <article className={`diagnostic-check ${check.status}`} key={check.id}>
                <span className="diagnostic-check-icon">{checkIcon(check.status)}</span>
                <div><div className="diagnostic-check-title"><strong>{check.label}</strong>{check.latencyMs !== undefined && <span>{check.latencyMs} ms</span>}</div><p>{check.detail}</p></div>
              </article>)}
            </div>
            <div className="diagnostic-limitations"><strong>此检测不代表以下内容</strong>{(report.limitations || []).map((item) => <p key={item}>{item}</p>)}</div>
            <div className="diagnostic-actions"><button className="button secondary" onClick={() => void run()} disabled={busy}>重新检测</button></div>
          </>}
        </div>
      </aside>
    </div>}
  </>
}

function formatLatency(value?: number): string { return value === undefined ? '—' : `${value} ms` }
function checkIcon(status: string): string { if (status === 'pass') return '✓'; if (status === 'warn') return '!'; if (status === 'fail') return '×'; return '–' }
function diagnosticStatusLabel(status: string): string { if (status === 'healthy') return '正常'; if (status === 'degraded') return '部分受限'; if (status === 'failed') return '失败'; return status }
