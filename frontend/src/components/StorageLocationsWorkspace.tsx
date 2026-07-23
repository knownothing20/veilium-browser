import { useEffect, useState } from 'react'
import {
  storageLocationAPI,
  type ManagedStorageLocation,
  type ManagedStorageLocations,
} from '../storageLocations'

export function StorageLocationsWorkspace() {
  const nativeMode = storageLocationAPI.isNative()
  const [data, setData] = useState<ManagedStorageLocations>()
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState('')
  const [error, setError] = useState('')

  const refresh = async () => {
    setLoading(true)
    setError('')
    try {
      setData(await storageLocationAPI.get())
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (nativeMode) void refresh()
  }, [nativeMode])

  const copyPath = async (item: ManagedStorageLocation) => {
    try {
      await navigator.clipboard.writeText(item.path)
      setCopied(item.id)
      window.setTimeout(() => setCopied((current) => current === item.id ? '' : current), 1600)
    } catch (reason) {
      setError(`Copy path failed: ${reason instanceof Error ? reason.message : String(reason)}`)
    }
  }

  return <section className="panel recovery-section">
    <div className="panel-heading">
      <div>
        <span className="eyebrow">Local-only managed paths</span>
        <h2>Storage locations</h2>
        <p>See exactly where Veilium keeps Profiles, browser Kernels, adapters, logs, lifecycle state, snapshots, recoverable trash, and templates. This view never moves or deletes data.</p>
      </div>
      <button className="button secondary" disabled={!nativeMode || loading} onClick={() => void refresh()}>{loading ? 'Refreshing…' : 'Refresh locations'}</button>
    </div>

    {!nativeMode && <div className="form-error">Managed storage locations require the Wails desktop runtime.</div>}
    {error && <div className="form-error">{error}</div>}

    {data && <>
      <div className={data.onSystemVolume ? 'form-error' : 'info-banner'}>
        <strong>{data.onSystemVolume ? 'Veilium data is on the Windows system volume' : 'Veilium data root is not on the detected Windows system volume'}</strong>
        <p>{data.onSystemVolume
          ? 'Profile browser data and managed browser packages can consume substantial space. Review the paths below before installing large packages.'
          : 'The paths below remain local to this installation. Portable artifacts and operation reports do not include these absolute paths.'}</p>
      </div>
      <dl>
        <div><dt>Data root</dt><dd><code>{data.dataRoot}</code></dd></div>
        <div><dt>Data volume</dt><dd>{data.dataVolume || 'Unknown'}</dd></div>
        <div><dt>Windows system volume</dt><dd>{data.systemVolume || 'Not detected on this platform'}</dd></div>
        <div><dt>Inspected</dt><dd>{formatTime(data.generatedAt)}</dd></div>
      </dl>

      <div className="recovery-card-grid">
        {(data.locations || []).map((item) => <article className="recovery-card" key={item.id}>
          <div className="recovery-card-head">
            <strong>{item.label}</strong>
            <span className={`lifecycle-operation-status ${statusClass(item)}`}>{statusLabel(item)}</span>
          </div>
          <p>{item.description}</p>
          <code title={item.path}>{item.path}</code>
          <dl>
            <div><dt>Expected type</dt><dd>{item.kind}</dd></div>
            <div><dt>Volume</dt><dd>{item.volume || 'Unknown'}</dd></div>
            <div><dt>System volume</dt><dd>{item.onSystemVolume ? 'Yes' : 'No'}</dd></div>
            {item.reasonCode && <div><dt>Finding</dt><dd>{item.reasonCode}</dd></div>}
          </dl>
          <button className="button secondary" disabled={loading} onClick={() => void copyPath(item)}>{copied === item.id ? 'Copied' : 'Copy path'}</button>
        </article>)}
      </div>

      <div className="info-banner">
        <strong>Safety boundaries</strong>
        <ul className="plain-list">{(data.limitations || []).map((item) => <li key={item}>{item}</li>)}</ul>
      </div>
    </>}
  </section>
}

function statusClass(item: ManagedStorageLocation): string {
  if (item.status === 'present') return 'passed'
  if (item.status === 'missing') return 'running'
  return 'failed'
}

function statusLabel(item: ManagedStorageLocation): string {
  if (item.status === 'present') return 'present'
  if (item.status === 'missing') return 'not created'
  if (item.status === 'unsafe-link') return 'unsafe link'
  if (item.status === 'unexpected-entry') return 'unexpected entry'
  return 'unavailable'
}

function formatTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
