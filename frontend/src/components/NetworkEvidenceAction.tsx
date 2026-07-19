import { useEffect, useMemo, useState } from 'react'
import { networkBackend } from '../lib/network-backend'
import type { NetworkEvidenceRun, NetworkProbeDefinition, NetworkProbeSet } from '../network-types'
import type { Profile, RuntimeSession } from '../types'

const defaultSet = (): NetworkProbeSet => ({
  schemaVersion: 1,
  id: 'local-network-probes',
  revision: 1,
  definitions: [
    { schemaVersion: 1, id: 'exit', revision: 1, kind: 'exit-ip', httpsUrl: '', timeoutSeconds: 10, maxResponseBytes: 4096, selfHostable: true, privacyNote: 'Returns only the public IP observed for this controlled browser request.' },
    { schemaVersion: 1, id: 'stun', revision: 1, kind: 'webrtc-stun', stunServer: '', timeoutSeconds: 10, selfHostable: true, privacyNote: 'Receives only the bounded STUN exchange for this controlled run.' },
    { schemaVersion: 1, id: 'dns', revision: 1, kind: 'delegated-dns', dnsZone: '', dnsResultUrl: '', timeoutSeconds: 10, maxResponseBytes: 4096, selfHostable: true, privacyNote: 'Records only the one-time delegated DNS query result.' },
  ],
})

export function NetworkEvidenceAction({ profile, session, nativeMode }: { profile: Profile; session?: RuntimeSession; nativeMode: boolean }) {
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [configured, setConfigured] = useState(false)
  const [probeSet, setProbeSet] = useState<NetworkProbeSet>(defaultSet)
  const [runs, setRuns] = useState<NetworkEvidenceRun[]>([])
  const [matrixCount, setMatrixCount] = useState<number>()
  const ready = session?.state === 'ready' && Boolean(session.cdpPort)
  const latest = runs[0]

  useEffect(() => {
    if (!open || !nativeMode) return
    void refresh()
  }, [open, nativeMode, profile.id])

  const definitions = useMemo(() => ({
    exit: probeSet.definitions.find((item) => item.kind === 'exit-ip'),
    stun: probeSet.definitions.find((item) => item.kind === 'webrtc-stun'),
    dns: probeSet.definitions.find((item) => item.kind === 'delegated-dns'),
  }), [probeSet])

  async function refresh() {
    setBusy(true)
    setError('')
    try {
      const [configuration, reports] = await Promise.all([
        networkBackend.configured(),
        networkBackend.list(profile.id),
      ])
      setConfigured(configuration.configured)
      setProbeSet(configuration.configured ? configuration.probeSet : defaultSet())
      setRuns(reports)
    } catch (reason) {
      setError(message(reason))
    } finally {
      setBusy(false)
    }
  }

  function updateDefinition(kind: NetworkProbeDefinition['kind'], patch: Partial<NetworkProbeDefinition>) {
    setProbeSet((current) => ({
      ...current,
      definitions: current.definitions.map((item) => item.kind === kind ? { ...item, ...patch } : item),
    }))
  }

  async function save() {
    setBusy(true)
    setError('')
    try {
      const saved = await networkBackend.saveProbeSet(probeSet)
      setProbeSet(saved)
      setConfigured(true)
    } catch (reason) {
      setError(message(reason))
    } finally {
      setBusy(false)
    }
  }

  async function removeConfig() {
    setBusy(true)
    setError('')
    try {
      await networkBackend.deleteProbeSet()
      setConfigured(false)
      setProbeSet(defaultSet())
    } catch (reason) {
      setError(message(reason))
    } finally {
      setBusy(false)
    }
  }

  async function run() {
    setBusy(true)
    setError('')
    try {
      await networkBackend.run(profile.id)
      setRuns(await networkBackend.list(profile.id))
    } catch (reason) {
      setError(message(reason))
      setRuns(await networkBackend.list(profile.id).catch(() => runs))
    } finally {
      setBusy(false)
    }
  }

  async function deleteRun(id: string) {
    setBusy(true)
    setError('')
    try {
      await networkBackend.delete(id)
      setRuns(await networkBackend.list(profile.id))
    } catch (reason) {
      setError(message(reason))
    } finally {
      setBusy(false)
    }
  }

  async function loadMatrix() {
    setBusy(true)
    setError('')
    try {
      const matrix = await networkBackend.matrix()
      setMatrixCount(matrix.entries.length)
    } catch (reason) {
      setError(message(reason))
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <button title={nativeMode ? 'Network evidence' : 'Desktop runtime required'} disabled={!nativeMode} onClick={() => setOpen(true)}>⌁✓</button>
      {open && (
        <div className="overlay diagnostic-overlay" onMouseDown={() => setOpen(false)}>
          <aside className="diagnostic-drawer" onMouseDown={(event) => event.stopPropagation()}>
            <header className="editor-header">
              <div><span className="eyebrow">M4.4 controlled browser probes</span><h2>Network evidence</h2><p>{profile.name}</p></div>
              <button className="close-button" onClick={() => setOpen(false)}>×</button>
            </header>
            <div className="diagnostic-content">
              {error && <div className="form-error">{error}</div>}
              <div className="diagnostic-limitations">
                <strong>Explicit probe configuration</strong>
                <p>Veilium has no hidden public probe. Configure HTTPS or self-hostable endpoints that permit CORS for the controlled browser page.</p>
              </div>
              <label>ProbeSet ID<input value={probeSet.id} onChange={(event) => setProbeSet({ ...probeSet, id: event.target.value })} /></label>
              <label>Revision<input type="number" min={1} value={probeSet.revision} onChange={(event) => setProbeSet({ ...probeSet, revision: Number(event.target.value) || 1 })} /></label>
              <label>Exit-IP HTTPS URL<input value={definitions.exit?.httpsUrl || ''} placeholder="https://probe.example/ip" onChange={(event) => updateDefinition('exit-ip', { httpsUrl: event.target.value })} /></label>
              <label>STUN server<input value={definitions.stun?.stunServer || ''} placeholder="stun:stun.example:3478" onChange={(event) => updateDefinition('webrtc-stun', { stunServer: event.target.value })} /></label>
              <label>Delegated DNS zone<input value={definitions.dns?.dnsZone || ''} placeholder="probe.example" onChange={(event) => updateDefinition('delegated-dns', { dnsZone: event.target.value })} /></label>
              <label>DNS result URL<input value={definitions.dns?.dnsResultUrl || ''} placeholder="https://probe.example/dns-result" onChange={(event) => updateDefinition('delegated-dns', { dnsResultUrl: event.target.value })} /></label>
              <div className="diagnostic-actions">
                <button className="button secondary" disabled={busy} onClick={() => void save()}>{configured ? 'Update ProbeSet' : 'Save ProbeSet'}</button>
                {configured && <button className="button secondary" disabled={busy} onClick={() => void removeConfig()}>Remove configuration</button>}
                <button className="button" disabled={busy || !configured || !ready} title={!ready ? 'Start the Profile and wait for Ready state' : ''} onClick={() => void run()}>{busy ? 'Running…' : 'Run network evidence'}</button>
              </div>
              {!ready && <div className="loading-block">The managed browser session must be running and Ready.</div>}
              {latest && (
                <>
                  <div className="diagnostic-summary">
                    <span className={`diagnostic-status ${latest.status}`}>{latest.status}</span>
                    <div><span>Route</span><strong>{latest.route.kind}</strong></div>
                    <div><span>ProbeSet</span><strong>{latest.probeSetId} r{latest.probeSetRevision}</strong></div>
                    <div><span>Expires</span><strong>{new Date(latest.expiresAt).toLocaleString()}</strong></div>
                  </div>
                  <div className="diagnostic-checks">
                    {latest.observations.map((item) => (
                      <article className={`diagnostic-check ${item.status}`} key={item.id}>
                        <span className="diagnostic-check-icon">{icon(item.status)}</span>
                        <div><div className="diagnostic-check-title"><strong>{item.probeKind}</strong><span>{item.status}</span></div><p>{item.detail || item.reasonCode || (item.values || []).join(', ')}</p></div>
                      </article>
                    ))}
                  </div>
                  {(latest.limitations || []).map((item) => <p key={item}>{item}</p>)}
                  <div className="diagnostic-actions"><button className="button secondary" disabled={busy} onClick={() => void deleteRun(latest.id)}>Delete latest report</button></div>
                </>
              )}
              <div className="diagnostic-actions"><button className="button secondary" disabled={busy} onClick={() => void loadMatrix()}>Generate compatibility matrix</button>{matrixCount !== undefined && <span>{matrixCount} exact entries</span>}</div>
            </div>
          </aside>
        </div>
      )}
    </>
  )
}

function message(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }
function icon(status: string): string { if (status === 'passed') return '✓'; if (status === 'failed') return '×'; if (status === 'partial') return '!'; return '–' }
