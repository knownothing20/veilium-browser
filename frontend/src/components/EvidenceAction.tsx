import { useMemo, useState } from 'react'
import { backend } from '../lib/backend'
import { evidenceStatusClass, evidenceStatusLabel, evidenceSummary, latestEvidence } from '../lib/evidence'
import type { EvidenceRun, Profile, RuntimeSession } from '../types'

export function EvidenceAction({
  profile,
  session,
  nativeMode,
}: {
  profile: Profile
  session?: RuntimeSession
  nativeMode: boolean
}) {
  const [open, setOpen] = useState(false)
  const [runs, setRuns] = useState<EvidenceRun[]>([])
  const [selectedID, setSelectedID] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const selected = useMemo(
    () => runs.find((run) => run.id === selectedID) || latestEvidence(runs),
    [runs, selectedID],
  )
  const canReview = nativeMode && Boolean(profile.kernel.id)
  const ready = canReview && session?.state === 'ready' && Boolean(session.cdpPort)

  async function load(clearError = true) {
    if (clearError) setError('')
    try {
      const items = await backend.listEvidence(profile.id)
      setRuns(items)
      if (!selectedID && items[0]) setSelectedID(items[0].id)
    } catch (reason) {
      setError(errorText(reason))
    }
  }

  async function show() {
    setOpen(true)
    await load()
  }

  async function runEvidence() {
    setBusy(true)
    setError('')
    try {
      const result = await backend.runEvidence(profile.id)
      await load(false)
      setSelectedID(result.id)
    } catch (reason) {
      await load(false)
      setError(errorText(reason))
    } finally {
      setBusy(false)
    }
  }

  async function cancel() {
    try {
      await backend.cancelEvidence(profile.id)
    } catch (reason) {
      setError(errorText(reason))
    }
  }

  async function remove(run: EvidenceRun) {
    if (!window.confirm('Delete this local evidence report?')) return
    try {
      await backend.deleteEvidence(run.id)
      const next = runs.filter((item) => item.id !== run.id)
      setRuns(next)
      setSelectedID(next[0]?.id || '')
    } catch (reason) {
      setError(errorText(reason))
    }
  }

  return (
    <>
      <button
        title={!canReview ? 'Managed desktop profile required' : ready ? 'Run or review local browser evidence' : 'Review local evidence reports; start the browser to collect a new report'}
        disabled={!canReview}
        onClick={() => void show()}
      >
        ◉
      </button>
      {open && (
        <div className="evidence-overlay" onMouseDown={() => setOpen(false)}>
          <section className="evidence-dialog" onMouseDown={(event) => event.stopPropagation()}>
            <header>
              <div>
                <span className="eyebrow">Controlled local browser observations</span>
                <h2>{profile.name} evidence</h2>
                <p>No cookies, browsing content, credentials, or remote probe data are collected.</p>
              </div>
              <button className="close-button" onClick={() => setOpen(false)}>×</button>
            </header>
            <div className="evidence-toolbar">
              <button className="button primary" disabled={!ready || busy} onClick={() => void runEvidence()}>
                {busy ? 'Collecting…' : 'Run evidence'}
              </button>
              {busy && <button className="button secondary" onClick={() => void cancel()}>Cancel</button>}
              <button className="button secondary" disabled={busy} onClick={() => void load()}>Refresh</button>
            </div>
            {!ready && <div className="info-banner"><strong>Review mode</strong><p>Start this managed browser profile to collect a new real-browser report.</p></div>}
            {error && <div className="form-error">{error}</div>}
            <div className="evidence-layout">
              <aside className="evidence-run-list">
                {runs.length === 0 && <p className="evidence-empty">No local evidence reports yet.</p>}
                {runs.map((run) => (
                  <button className={selected?.id === run.id ? 'selected' : ''} key={run.id} onClick={() => setSelectedID(run.id)}>
                    <strong>{evidenceStatusLabel(run.status)}</strong>
                    <span>{new Date(run.startedAt).toLocaleString()}</span>
                    <small>{run.providerId} · {evidenceSummary(run)}</small>
                  </button>
                ))}
              </aside>
              <div className="evidence-report">
                {!selected ? <p className="evidence-empty">Select or run an evidence report.</p> : (
                  <>
                    <div className="evidence-report-head">
                      <div>
                        <span className={`status-pill ${evidenceStatusClass(selected.status)}`}>{evidenceStatusLabel(selected.status)}</span>
                        <h3>{evidenceSummary(selected)}</h3>
                        <p>{selected.providerId} rev {selected.providerRevision} · {selected.providerTrust} · Chromium {selected.browserVersion}</p>
                      </div>
                      <button className="button secondary danger-text" onClick={() => void remove(selected)}>Delete report</button>
                    </div>
                    {selected.failureDetail && <div className="evidence-failure"><strong>{selected.failureCode || 'collection-failed'}</strong><p>{selected.failureDetail}</p></div>}
                    {selected.limitations?.length ? <ul className="evidence-limitations">{selected.limitations.map((item) => <li key={item}>{item}</li>)}</ul> : null}
                    <div className="evidence-observations">
                      {(selected.observations || []).map((observation) => (
                        <article key={`${observation.context}-${observation.id}`}>
                          <div><strong>{observation.id}</strong><span className={`observation-status ${observation.status}`}>{observation.status}</span></div>
                          <small>{observation.context}{observation.capabilityId ? ` · ${observation.capabilityId}` : ''}</small>
                          {observation.expected && <p><b>Expected:</b> {observation.expected}</p>}
                          {observation.observed && <p><b>Observed:</b> {observation.observed}</p>}
                          {(observation.reasonCode || observation.detail) && <p className="observation-detail">{observation.reasonCode}{observation.detail ? ` — ${observation.detail}` : ''}</p>}
                        </article>
                      ))}
                    </div>
                  </>
                )}
              </div>
            </div>
          </section>
        </div>
      )}
    </>
  )
}

function errorText(reason: unknown): string {
  return reason instanceof Error ? reason.message : String(reason)
}
