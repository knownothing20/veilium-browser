import { useEffect, useMemo, useState, type ChangeEvent } from 'react'
import { lifecycleRecordFor } from '../lifecycle'
import type { RecoveryWorkspaceData } from '../localRecovery'
import {
  multiProfileAPI,
  newMultiProfileKey,
  type BulkHealthRefreshResult,
  type ProfileHealthReport,
  type StorageManagementState,
} from '../multiProfile'

export function MultiProfileWorkspace({ data, onRefresh }: { data: RecoveryWorkspaceData; onRefresh: () => Promise<void> }) {
  const nativeMode = multiProfileAPI.isNative()
  const [selected, setSelected] = useState<string[]>([])
  const [setGroup, setSetGroup] = useState(false)
  const [group, setGroupValue] = useState('')
  const [addTags, setAddTags] = useState('')
  const [removeTags, setRemoveTags] = useState('')
  const [health, setHealth] = useState<BulkHealthRefreshResult>()
  const [storage, setStorage] = useState<StorageManagementState>()
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const eligible = useMemo(() => data.profiles.filter((profile) => {
    const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
    const active = data.sessions.some((session) => session.profileId === profile.id && ['starting', 'ready', 'stopping'].includes(session.state))
    return Boolean(record && ['available', 'draft'].includes(record.state) && !record.lock && !active)
  }), [data.profiles, data.lifecycleRecords, data.sessions])

  useEffect(() => {
    const known = new Set(data.profiles.map((profile) => profile.id))
    setSelected((current) => current.filter((profileId) => known.has(profileId)))
  }, [data.profiles])

  const run = async (key: string, action: () => Promise<void>) => {
    setBusy(key)
    setError('')
    setNotice('')
    try { await action() }
    catch (reason) { setError(errorText(reason)) }
    finally { setBusy('') }
  }

  const toggle = (profileId: string) => {
    setSelected((current) => current.includes(profileId)
      ? current.filter((item) => item !== profileId)
      : [...current, profileId])
  }

  const updateMetadata = () => run('metadata', async () => {
    if (selected.length === 0) throw new Error('Select at least one eligible Profile.')
    const added = splitTags(addTags)
    const removed = splitTags(removeTags)
    if (!setGroup && added.length === 0 && removed.length === 0) throw new Error('Choose a group or tag change.')
    const result = await multiProfileAPI.updateMetadata({
      profileIds: selected,
      setGroup,
      group,
      addTags: added,
      removeTags: removed,
      idempotencyKey: newMultiProfileKey(),
    })
    const succeeded = result.operation.items?.filter((item) => item.status === 'succeeded').length || 0
    const skipped = result.operation.items?.filter((item) => item.status !== 'succeeded').length || 0
    setNotice(`Bulk metadata ${result.operation.status}: ${succeeded} succeeded${skipped ? `, ${skipped} not changed` : ''}.`)
    setAddTags('')
    setRemoveTags('')
    await onRefresh()
  })

  const refreshHealth = () => run('health', async () => {
    if (selected.length === 0) throw new Error('Select at least one eligible Profile.')
    const result = await multiProfileAPI.refreshHealth({
      profileIds: selected,
      idempotencyKey: newMultiProfileKey(),
    })
    setHealth(result)
    const ready = result.reports.filter((item) => item.status === 'ready').length
    const limited = result.reports.filter((item) => item.status === 'limited').length
    const blocked = result.reports.filter((item) => item.status === 'blocked').length
    setNotice(`Health refresh ${result.operation.status}: ${ready} ready, ${limited} limited, ${blocked} blocked.`)
    await onRefresh()
  })

  const refreshStorage = () => run('storage', async () => {
    const result = await multiProfileAPI.refreshStorage()
    setStorage(result)
    setNotice(`Storage inventory refreshed for ${result.inventory.profiles.length} Profile records.`)
  })

  return <section className="panel recovery-section">
    <div className="panel-heading"><div><h2>Multi-Profile and storage management</h2><p>Apply bounded changes to a fixed Profile selection, refresh local launch health, and inspect managed storage without automatic cleanup or repair.</p></div></div>
    {!nativeMode && <div className="form-error">Multi-Profile and storage actions require the Wails desktop runtime.</div>}
    {error && <div className="form-error">{error}</div>}
    {notice && <div className="info-banner"><strong>Completed</strong><p>{notice}</p></div>}

    <div className="settings-grid">
      <article className="panel setting-card">
        <div className="panel-heading"><div><h2>Fixed Profile selection</h2><p>Only available or draft Profiles with no active browser or lifecycle lock can be selected.</p></div><span className="lifecycle-operation-status running">{selected.length} selected</span></div>
        <div className="toolbar">
          <button className="button secondary" disabled={Boolean(busy) || eligible.length === 0} onClick={() => setSelected(eligible.map((item) => item.id))}>Select eligible</button>
          <button className="button secondary" disabled={Boolean(busy) || selected.length === 0} onClick={() => setSelected([])}>Clear</button>
        </div>
        <div className="recovery-profile-list">
          {data.profiles.length === 0 ? <div className="lifecycle-empty">No Profiles are available.</div> : data.profiles.map((profile) => {
            const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
            const active = data.sessions.some((session) => session.profileId === profile.id && ['starting', 'ready', 'stopping'].includes(session.state))
            const allowed = Boolean(record && ['available', 'draft'].includes(record.state) && !record.lock && !active)
            return <label className="recovery-row" key={profile.id}>
              <input type="checkbox" checked={selected.includes(profile.id)} disabled={!allowed || Boolean(busy)} onChange={() => toggle(profile.id)} />
              <div className="recovery-identity"><div className="avatar">{profile.name.slice(0, 2).toUpperCase()}</div><div><strong>{profile.name}</strong><span>{profile.group || 'No group'} · {(profile.tags || []).join(', ') || 'No tags'}</span></div></div>
              <div className="recovery-state"><span className={`lifecycle-pill ${record?.state || 'missing'} ${record?.lock ? 'locked' : ''}`}>{record?.state || 'missing'}</span><small>{active ? 'Browser active' : record?.lock ? 'Lifecycle lock active' : allowed ? 'Eligible' : 'Not eligible'}</small></div>
            </label>
          })}
        </div>
      </article>

      <article className="panel setting-card">
        <div className="panel-heading"><div><h2>Bulk metadata</h2><p>Replace a group and add or remove bounded tags without changing browser data, routes, fingerprints, or credentials.</p></div></div>
        <label className="checkbox-line"><input type="checkbox" checked={setGroup} onChange={(event: ChangeEvent<HTMLInputElement>) => setSetGroup(event.target.checked)} /><span>Replace group for selected Profiles</span></label>
        <label>Group<input value={group} disabled={!setGroup} maxLength={128} onChange={(event: ChangeEvent<HTMLInputElement>) => setGroupValue(event.target.value)} placeholder="Blank clears the group" /></label>
        <label>Add tags<input value={addTags} onChange={(event: ChangeEvent<HTMLInputElement>) => setAddTags(event.target.value)} placeholder="Comma-separated tags" /></label>
        <label>Remove tags<input value={removeTags} onChange={(event: ChangeEvent<HTMLInputElement>) => setRemoveTags(event.target.value)} placeholder="Comma-separated tags" /></label>
        <button className="button primary" disabled={!nativeMode || selected.length === 0 || Boolean(busy)} onClick={() => void updateMetadata()}>{busy === 'metadata' ? 'Updating…' : 'Apply bounded metadata update'}</button>
      </article>

      <article className="panel setting-card bulk-health-card">
        <div className="panel-heading"><div><h2>Bulk health refresh</h2><p>Revalidate lifecycle, managed Kernel integrity, route dependencies, fingerprint policy, identity consistency, and managed browser-data containment.</p></div><button className="button secondary" disabled={!nativeMode || selected.length === 0 || Boolean(busy)} onClick={() => void refreshHealth()}>{busy === 'health' ? 'Refreshing…' : 'Refresh selected health'}</button></div>
        {!health ? <div className="lifecycle-empty">Select stopped Profiles and run a read-only health refresh.</div> : <div className="bulk-health-list">
          {health.reports.map((report) => <HealthReport key={report.profileId} report={report} />)}
        </div>}
      </article>

      <article className="panel setting-card">
        <div className="panel-heading"><div><h2>Managed storage inventory</h2><p>Counts opaque Profile files and reports missing, orphaned, unsafe, or incomplete entries. It never deletes data.</p></div><button className="button secondary" disabled={!nativeMode || Boolean(busy)} onClick={() => void refreshStorage()}>{busy === 'storage' ? 'Scanning…' : 'Refresh inventory'}</button></div>
        {!storage ? <div className="lifecycle-empty">Run a storage scan to inspect current managed Profile data.</div> : <>
          <dl>
            <div><dt>Profile storage</dt><dd>{storage.inventory.profiles.length} records · {formatBytes(storage.inventory.summary.bytes)}</dd></div>
            <div><dt>Verified snapshots</dt><dd>{storage.snapshotCount} · {formatBytes(storage.snapshotBytes)}</dd></div>
            <div><dt>Recoverable trash</dt><dd>{storage.trashCount} · {formatBytes(storage.trashBytes)}</dd></div>
            <div><dt>Lifecycle history</dt><dd>{storage.operationCount} operations</dd></div>
            <div><dt>Scan status</dt><dd>{storage.inventory.incomplete ? 'Incomplete' : 'Complete'} · {formatTime(storage.generatedAt)}</dd></div>
          </dl>
          <div className="recovery-profile-list">
            {storage.inventory.profiles.map((item) => <article className="recovery-row" key={item.profileId}>
              <div className="recovery-identity"><div><strong>{profileName(data, item.profileId)}</strong><span>{item.managedDir}</span></div></div>
              <div className="recovery-state"><span className={`lifecycle-operation-status ${item.status}`}>{item.status}</span><small>{item.summary.files} files · {formatBytes(item.summary.bytes)}{item.reasonCode ? ` · ${item.reasonCode}` : ''}</small></div>
            </article>)}
          </div>
          {Boolean(storage.inventory.orphans?.length || storage.inventory.unsafe?.length) && <ul className="lifecycle-findings">
            {storage.inventory.orphans?.map((item) => <li className="warn" key={`orphan-${item.relativePath}`}><strong>Orphan candidate</strong><span>{item.relativePath} · {item.reasonCode}</span></li>)}
            {storage.inventory.unsafe?.map((item) => <li className="danger" key={`unsafe-${item.relativePath}`}><strong>Unsafe entry</strong><span>{item.relativePath} · {item.reasonCode}</span></li>)}
          </ul>}
          <ul className="plain-list">{storage.limitations?.map((item) => <li key={item}>{item}</li>)}</ul>
        </>}
      </article>
    </div>
  </section>
}

function HealthReport({ report }: { report: ProfileHealthReport }) {
  return <article className={`bulk-health-report ${report.status}`}>
    <div className="bulk-health-report-header">
      <div><strong>{report.profileName}</strong><span>{report.lifecycleState} · {formatTime(report.refreshedAt)}</span></div>
      <span className={`bulk-health-status ${report.status}`}>{report.status}</span>
    </div>
    <ul className="bulk-health-checks">
      {report.checks.map((check) => <li className={check.status} key={check.id}>
        <span className="bulk-health-check-icon">{check.status === 'pass' ? '✓' : check.status === 'warning' ? '!' : '×'}</span>
        <div><strong>{healthCheckLabel(check.id)}</strong><p>{check.message}</p></div>
      </li>)}
    </ul>
  </article>
}

function healthCheckLabel(id: string): string {
  switch (id) {
    case 'lifecycle': return 'Lifecycle'
    case 'kernel': return 'Managed Kernel'
    case 'route': return 'Route and credentials'
    case 'fingerprint': return 'Fingerprint policy'
    case 'consistency': return 'Identity consistency'
    case 'managed-data': return 'Managed browser data'
    default: return id
  }
}

function splitTags(value: string): string[] {
  return value.split(/[\n,]/).map((item) => item.trim()).filter(Boolean)
}

function profileName(data: RecoveryWorkspaceData, profileId: string): string {
  return data.profiles.find((item) => item.id === profileId)?.name || profileId
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1)
  return `${(value / (1024 ** index)).toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

function formatTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function errorText(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }
