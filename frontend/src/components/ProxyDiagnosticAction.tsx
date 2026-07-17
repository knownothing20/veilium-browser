import { useState } from 'react'
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
    try {
      setReport(await backend.runProxyDiagnostics(profile.id))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <button
        title={nativeMode ? 'Test proxy route' : 'Desktop runtime required'}
        disabled={!nativeMode || busy}
        onClick={() => void run()}
      >
        {busy ? '…' : '⌁'}
      </button>
      {open && (
        <div className="overlay diagnostic-overlay" onMouseDown={() => setOpen(false)}>
          <aside className="diagnostic-drawer" onMouseDown={(event) => event.stopPropagation()}>
            <header className="editor-header">
              <div>
                <span className="eyebrow">Live route measurement</span>
                <h2>Proxy diagnostics</h2>
                <p>{profile.name}</p>
              </div>
              <button className="close-button" onClick={() => setOpen(false)}>×</button>
            </header>
            <div className="diagnostic-content">
              {busy && <div className="loading-block">Testing the selected network route…</div>}
              {error && <div className="form-error">{error}</div>}
              {report && !busy && (
                <>
                  <div className="diagnostic-summary">
                    <span className={`diagnostic-status ${report.status}`}>{report.status}</span>
                    <div><span>Exit IP</span><strong>{report.exitIp || 'Unavailable'}</strong></div>
                    <div><span>First byte</span><strong>{formatLatency(report.firstByteLatencyMs)}</strong></div>
                    <div><span>Total</span><strong>{formatLatency(report.totalLatencyMs)}</strong></div>
                  </div>
                  <dl className="diagnostic-route">
                    <div><dt>Configured route</dt><dd>{report.proxyDisplay}</dd></div>
                    <div><dt>Route kind</dt><dd>{report.routeKind}</dd></div>
                    <div><dt>Temporary bridge</dt><dd>{report.bridgeKind || 'Not required'}</dd></div>
                    <div><dt>Completed</dt><dd>{new Date(report.completedAt).toLocaleString()}</dd></div>
                  </dl>
                  <div className="diagnostic-checks">
                    {report.checks.map((check) => (
                      <article className={`diagnostic-check ${check.status}`} key={check.id}>
                        <span className="diagnostic-check-icon">{checkIcon(check.status)}</span>
                        <div>
                          <div className="diagnostic-check-title">
                            <strong>{check.label}</strong>
                            {check.latencyMs !== undefined && <span>{check.latencyMs} ms</span>}
                          </div>
                          <p>{check.detail}</p>
                        </div>
                      </article>
                    ))}
                  </div>
                  <div className="diagnostic-limitations">
                    <strong>What this test does not claim</strong>
                    {report.limitations.map((item) => <p key={item}>{item}</p>)}
                  </div>
                  <div className="diagnostic-actions">
                    <button className="button secondary" onClick={() => void run()} disabled={busy}>Run again</button>
                  </div>
                </>
              )}
            </div>
          </aside>
        </div>
      )}
    </>
  )
}

function formatLatency(value?: number): string {
  return value === undefined ? '—' : `${value} ms`
}

function checkIcon(status: string): string {
  if (status === 'pass') return '✓'
  if (status === 'warn') return '!'
  if (status === 'fail') return '×'
  return '–'
}
