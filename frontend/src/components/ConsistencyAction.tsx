import { useState } from 'react'
import type { Profile } from '../types'

type HealthStatus = 'healthy' | 'degraded' | 'blocked' | 'unknown'
type CheckStatus = 'passed' | 'warning' | 'failed' | 'unknown'

interface ConsistencyCheck {
  id: string
  status: CheckStatus
  expected?: string
  observed?: string
  reasonCode?: string
  detail?: string
}

interface ConsistencyResult {
  schemaVersion: number
  rulesRevision: string
  profileId: string
  inputDigest: string
  evidenceRunId?: string
  evidenceFresh: boolean
  status: HealthStatus
  window: {
    width: number
    height: number
    deviceScaleFactor: number
    source: 'explicit' | 'legacy-screen-fallback'
  }
  checks: ConsistencyCheck[]
  blockingReasons?: string[]
  degradedReasons?: string[]
  generatedAt: string
}

type ConsistencyAPI = {
  ProfileConsistency: (profileId: string) => Promise<ConsistencyResult>
}

function api(): ConsistencyAPI | undefined {
  const value = window as Window & {
    go?: { main?: { DesktopApp?: ConsistencyAPI } }
  }
  return value.go?.main?.DesktopApp
}

export function ConsistencyAction({ profile, nativeMode }: { profile: Profile; nativeMode: boolean }) {
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<ConsistencyResult>()

  async function inspect() {
    setOpen(true)
    setLoading(true)
    setError('')
    try {
      const desktop = api()
      if (!desktop) throw new Error('Profile consistency is available only in the desktop application')
      setResult(await desktop.ProfileConsistency(profile.id))
    } catch (reason) {
      setResult(undefined)
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <button
        title={nativeMode ? 'Inspect identity and window consistency' : 'Desktop runtime required'}
        disabled={!nativeMode || !profile.kernel.id}
        onClick={() => void inspect()}
      >
        ◈
      </button>
      {open && (
        <div className="overlay" onMouseDown={() => setOpen(false)}>
          <section className="evidence-dialog" onMouseDown={(event) => event.stopPropagation()}>
            <header className="editor-header">
              <div>
                <span className="eyebrow">Evidence-derived health</span>
                <h2>{profile.name} consistency</h2>
              </div>
              <button type="button" className="close-button" onClick={() => setOpen(false)}>×</button>
            </header>
            <div className="editor-scroll evidence-report-scroll">
              {loading && <div className="info-banner"><strong>Evaluating current profile…</strong></div>}
              {error && <div className="form-error">{error}</div>}
              {result && (
                <>
                  <div className="evidence-summary-grid">
                    <div><span>Status</span><strong className={`status-pill ${result.status}`}>{result.status}</strong></div>
                    <div><span>Window</span><strong>{result.window.width} × {result.window.height}</strong></div>
                    <div><span>DPR</span><strong>{result.window.deviceScaleFactor}</strong></div>
                    <div><span>Evidence</span><strong>{result.evidenceFresh ? 'fresh' : 'missing or stale'}</strong></div>
                  </div>
                  <div className="info-banner">
                    <strong>{result.window.source === 'explicit' ? 'Explicit window plan' : 'Legacy compatibility fallback'}</strong>
                    <p>Rules {result.rulesRevision} · generated {new Date(result.generatedAt).toLocaleString()}</p>
                  </div>
                  {(result.blockingReasons?.length || 0) > 0 && (
                    <div className="form-error">
                      <strong>Launch-blocking reasons</strong>
                      <p>{result.blockingReasons!.join(' · ')}</p>
                    </div>
                  )}
                  {(result.degradedReasons?.length || 0) > 0 && (
                    <div className="info-banner">
                      <strong>Degraded or unknown reasons</strong>
                      <p>{result.degradedReasons!.join(' · ')}</p>
                    </div>
                  )}
                  <div className="evidence-observation-list">
                    {result.checks.map((check) => (
                      <article key={check.id} className={`evidence-observation ${check.status}`}>
                        <div className="evidence-observation-head">
                          <strong>{check.id}</strong>
                          <span>{check.status}</span>
                        </div>
                        {(check.expected || check.observed) && (
                          <dl>
                            {check.expected && <div><dt>Expected</dt><dd>{check.expected}</dd></div>}
                            {check.observed && <div><dt>Observed</dt><dd>{check.observed}</dd></div>}
                          </dl>
                        )}
                        {(check.detail || check.reasonCode) && <p>{check.detail || check.reasonCode}</p>}
                      </article>
                    ))}
                  </div>
                </>
              )}
            </div>
            <footer className="editor-footer">
              <button type="button" className="button secondary" onClick={() => void inspect()} disabled={loading}>Refresh</button>
              <button type="button" className="button primary" onClick={() => setOpen(false)}>Close</button>
            </footer>
          </section>
        </div>
      )}
    </>
  )
}
