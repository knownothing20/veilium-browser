import { useEffect, useMemo, useState } from 'react'
import { cancellationAvailability, lifecycleRecordFor } from '../lifecycle'
import {
  localRecoveryAPI,
  newRecoveryKey,
  type LocalRecoveryPreflight,
  type LocalRecoveryState,
  type RecoveryWorkspaceData,
} from '../localRecovery'

export function RecoveryWorkspace({ data, onRefresh }: { data: RecoveryWorkspaceData; onRefresh: () => Promise<void> }) {
  const [state, setState] = useState<LocalRecoveryState>(() => localRecoveryAPI.emptyState())
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const nativeMode = localRecoveryAPI.isNative()

  const refresh = async () => {
    try {
      setState(await localRecoveryAPI.state())
    } catch (reason) {
      setError(errorText(reason))
    }
  }

  useEffect(() => {
    void refresh()
    if (!nativeMode) return
    const timer = window.setInterval(() => void refresh(), 1500)
    return () => window.clearInterval(timer)
  }, [nativeMode])

  const operations = useMemo(() => [...data.lifecycleOperations]
    .filter((item) => ['snapshot', 'restore', 'archive', 'unarchive', 'trash', 'restore-trash', 'permanent-delete'].includes(item.type))
    .sort((left, right) => right.updatedAt.localeCompare(left.updatedAt)), [data.lifecycleOperations])

  const run = async (key: string, action: () => Promise<LocalRecoveryState>) => {
    setBusy(key)
    setError('')
    try {
      setState(await action())
      await onRefresh()
    } catch (reason) {
      setError(errorText(reason))
      await refresh()
      await onRefresh()
    } finally {
      setBusy('')
    }
  }

  const requirePreflight = async (profileId: string, allowed: keyof Pick<LocalRecoveryPreflight, 'snapshotAllowed' | 'archiveAllowed' | 'unarchiveAllowed' | 'trashAllowed'>) => {
    const result = await localRecoveryAPI.preflight(profileId)
    if (!result[allowed]) throw new Error(result.reasons?.join(' · ') || `Local recovery preflight rejected ${allowed}.`)
  }

  const snapshot = (profileId: string) => run(`snapshot-${profileId}`, async () => {
    await requirePreflight(profileId, 'snapshotAllowed')
    return localRecoveryAPI.snapshot({ profileId, idempotencyKey: newRecoveryKey() })
  })

  const archive = (profileId: string) => run(`archive-${profileId}`, async () => {
    await requirePreflight(profileId, 'archiveAllowed')
    return localRecoveryAPI.archive({ profileId, idempotencyKey: newRecoveryKey() })
  })

  const unarchive = (profileId: string) => run(`unarchive-${profileId}`, async () => {
    await requirePreflight(profileId, 'unarchiveAllowed')
    return localRecoveryAPI.unarchive({ profileId, idempotencyKey: newRecoveryKey() })
  })

  const trash = (profileId: string) => run(`trash-${profileId}`, async () => {
    await requirePreflight(profileId, 'trashAllowed')
    if (!window.confirm('Move this Profile and its managed browser data to recoverable trash for 30 days?')) throw new Error('Trash operation cancelled.')
    return localRecoveryAPI.trash({ profileId, retentionDays: 30, idempotencyKey: newRecoveryKey() })
  })

  const restoreSnapshot = (snapshotId: string) => run(`restore-${snapshotId}`, async () => {
    const name = window.prompt('Optional name for the restored Profile:', '') || ''
    return localRecoveryAPI.restoreSnapshot({ snapshotId, name, idempotencyKey: newRecoveryKey() })
  })

  const restoreTrash = (profileId: string, trashId: string) => run(`restore-trash-${trashId}`, () =>
    localRecoveryAPI.restoreTrash({ profileId, trashId, idempotencyKey: newRecoveryKey() }))

  const permanentDelete = (profileId: string, trashId: string) => run(`delete-${trashId}`, async () => {
    const confirmation = window.prompt(`Permanent deletion cannot be undone. Type the Profile ID exactly:\n${profileId}`, '') || ''
    if (confirmation !== profileId) throw new Error('Permanent deletion confirmation did not match the Profile ID.')
    return localRecoveryAPI.permanentDelete({ profileId, trashId, confirmation, idempotencyKey: newRecoveryKey() })
  })

  const cancel = (operationId: string) => run(`cancel-${operationId}`, () => localRecoveryAPI.cancel(operationId))
  const manualRefresh = () => run('refresh', () => localRecoveryAPI.refresh())

  return (
    <>
      <div className="page-heading compact">
        <div><span className="eyebrow">M5.2 local lifecycle storage</span><h1>Local recovery</h1><p>Create verified same-machine snapshots, restore to a new identity, archive Profiles, and manage recoverable trash.</p></div>
        <button className="button secondary" disabled={!nativeMode || Boolean(busy)} onClick={() => void manualRefresh()}>{busy === 'refresh' ? 'Checking…' : 'Refresh state'}</button>
      </div>
      {!nativeMode && <div className="form-error">Local recovery actions require the Wails desktop runtime.</div>}
      {error && <div className="form-error">{error}</div>}

      <section className="panel recovery-summary">
        <Summary label="Verified snapshots" value={state.snapshots.filter((item) => item.status === 'verified').length} detail={`${state.snapshots.length} catalog records`} />
        <Summary label="Recoverable trash" value={state.trash.filter((item) => item.status === 'stored').length} detail={`${state.trash.length} retained records`} />
        <Summary label="Running" value={state.progress.filter((item) => item.status === 'running' || item.status === 'pending').length} detail="Bounded local operations" />
        <Summary label="Recovery findings" value={state.trashReconciliation.findings?.length || 0} detail="No authority is guessed" warn={Boolean(state.trashReconciliation.findings?.length)} />
      </section>

      <section className="panel recovery-section">
        <div className="panel-heading"><div><h2>Profile actions</h2><p>Actions are restricted by lifecycle state, active sessions, locks, and storage preflight.</p></div></div>
        <div className="recovery-profile-list">
          {data.profiles.length === 0 ? <Empty text="No Profiles are available." /> : data.profiles.map((profile) => {
            const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
            const active = data.sessions.some((item) => item.profileId === profile.id && ['starting', 'ready', 'stopping'].includes(item.state))
            const locked = Boolean(record?.lock)
            const disabled = !nativeMode || active || locked || Boolean(busy)
            return <article className="recovery-row" key={profile.id}>
              <div className="recovery-identity"><div className="avatar">{profile.name.slice(0, 2).toUpperCase()}</div><div><strong>{profile.name}</strong><span>{profile.id}</span></div></div>
              <div className="recovery-state"><span className={`lifecycle-pill ${record?.state || 'missing'} ${locked ? 'locked' : ''}`}>{record?.state || 'missing'}</span><small>{active ? 'Browser session active' : locked ? `Locked by ${record?.lock?.operationId}` : 'No active blocker'}</small></div>
              <div className="recovery-actions">
                {record?.state === 'available' && <button className="button secondary" disabled={disabled} onClick={() => void snapshot(profile.id)}>{busy === `snapshot-${profile.id}` ? 'Snapshotting…' : 'Snapshot'}</button>}
                {(record?.state === 'available' || record?.state === 'draft') && <button className="button secondary" disabled={disabled} onClick={() => void archive(profile.id)}>{busy === `archive-${profile.id}` ? 'Archiving…' : 'Archive'}</button>}
                {record?.state === 'archived' && <button className="button secondary" disabled={disabled} onClick={() => void unarchive(profile.id)}>{busy === `unarchive-${profile.id}` ? 'Restoring state…' : 'Unarchive'}</button>}
                {record && ['available', 'draft', 'archived'].includes(record.state) && <button className="button secondary danger-text" disabled={disabled} onClick={() => void trash(profile.id)}>{busy === `trash-${profile.id}` ? 'Moving…' : 'Move to trash'}</button>}
              </div>
            </article>
          })}
        </div>
      </section>

      <section className="panel recovery-section">
        <div className="panel-heading"><div><h2>Local snapshots</h2><p>Snapshots are same-user, same-machine recovery artifacts and never overwrite the source Profile.</p></div></div>
        <div className="recovery-card-grid">
          {state.snapshots.length === 0 ? <Empty text="No local snapshot has been created." /> : [...state.snapshots].reverse().map((item) => <article className="recovery-card" key={item.snapshotId}>
            <div className="recovery-card-head"><strong>{profileName(data, item.sourceProfileId)}</strong><span className={`lifecycle-operation-status ${item.status}`}>{item.status}</span></div>
            <code>{item.snapshotId}</code>
            <dl><div><dt>Created</dt><dd>{formatTime(item.createdAt)}</dd></div><div><dt>Payload</dt><dd>{item.fileCount} files · {formatBytes(item.totalBytes)}</dd></div><div><dt>Tree identity</dt><dd>{item.treeDigest.slice(0, 16)}…</dd></div></dl>
            <button className="button primary" disabled={!nativeMode || item.status !== 'verified' || Boolean(busy)} onClick={() => void restoreSnapshot(item.snapshotId)}>{busy === `restore-${item.snapshotId}` ? 'Restoring…' : 'Restore to new identity'}</button>
          </article>)}
        </div>
      </section>

      <section className="panel recovery-section">
        <div className="panel-heading"><div><h2>Recoverable trash</h2><p>Retention is visible metadata only. Permanent deletion always requires the exact Profile ID.</p></div></div>
        <div className="recovery-card-grid">
          {state.trash.filter((item) => item.status !== 'deleted').length === 0 ? <Empty text="Recoverable trash is empty." /> : state.trash.filter((item) => item.status !== 'deleted').map((item) => <article className="recovery-card" key={item.trashId}>
            <div className="recovery-card-head"><strong>{profileName(data, item.profileId)}</strong><span className={`lifecycle-operation-status ${item.status}`}>{item.status}</span></div>
            <code>{item.trashId}</code>
            <dl><div><dt>Original state</dt><dd>{item.originalState}</dd></div><div><dt>Payload</dt><dd>{item.fileCount} files · {formatBytes(item.totalBytes)}</dd></div><div><dt>Retention</dt><dd>{formatTime(item.retentionDeadline)}</dd></div></dl>
            <div className="recovery-card-actions"><button className="button primary" disabled={!nativeMode || item.status !== 'stored' || !item.dataPresent || Boolean(busy)} onClick={() => void restoreTrash(item.profileId, item.trashId)}>{busy === `restore-trash-${item.trashId}` ? 'Restoring…' : 'Restore Profile'}</button><button className="button secondary danger-text" disabled={!nativeMode || item.status !== 'stored' || Boolean(busy)} onClick={() => void permanentDelete(item.profileId, item.trashId)}>{busy === `delete-${item.trashId}` ? 'Deleting…' : 'Delete permanently'}</button></div>
          </article>)}
        </div>
      </section>

      <section className="panel recovery-section">
        <div className="panel-heading"><div><h2>Operation progress and history</h2><p>Cancellation is only requested at declared safe boundaries.</p></div></div>
        <div className="recovery-operation-list">
          {state.progress.length === 0 && operations.length === 0 ? <Empty text="No local recovery operation has been recorded." /> : state.progress.map((item) => <article className="recovery-operation" key={item.operationId}>
            <div><strong>{label(item.operationType)}</strong><span>{item.operationId}</span></div>
            <div className="recovery-progress"><div><span style={{ width: `${progressPercent(item.bytesProcessed, item.bytesTotal)}%` }} /></div><small>{label(item.stage)} · {formatBytes(item.bytesProcessed)} / {formatBytes(item.bytesTotal)}</small></div>
            <span className={`lifecycle-operation-status ${item.status}`}>{item.status}</span>
            <button className="button secondary" disabled={!item.cancellationAvailable || Boolean(busy)} onClick={() => void cancel(item.operationId)}>Cancel</button>
          </article>)}
          {state.progress.length === 0 && operations.slice(0, 8).map((item) => <article className="recovery-operation" key={item.id}><div><strong>{label(item.type)}</strong><span>{item.id}</span></div><div><strong>{label(item.stage)}</strong><small>{cancellationAvailability(item)}</small></div><span className={`lifecycle-operation-status ${item.status}`}>{item.status}</span></article>)}
        </div>
      </section>

      <section className="panel recovery-section">
        <div className="panel-heading"><div><h2>Recovery-required state</h2><p>Interrupted or contradictory storage is preserved for manual review; the application does not guess which copy is authoritative.</p></div></div>
        {(state.trashReconciliation.findings?.length || 0) === 0 ? <Empty text="No contradictory trash state was found." /> : <ul className="lifecycle-findings">{state.trashReconciliation.findings?.map((item) => <li className="warn" key={`${item.trashId}-${item.reasonCode}`}><strong>{item.reasonCode}</strong><span>{item.profileId} · source {item.sourceState} · trash {item.trashState} · metadata {item.profileState}</span></li>)}</ul>}
      </section>
    </>
  )
}

function Summary({ label: title, value, detail, warn = false }: { label: string; value: number; detail: string; warn?: boolean }) {
  return <div className={`recovery-summary-item ${warn ? 'warn' : ''}`}><span>{title}</span><strong>{value}</strong><small>{detail}</small></div>
}
function Empty({ text }: { text: string }) { return <div className="lifecycle-empty">{text}</div> }
function profileName(data: RecoveryWorkspaceData, id: string): string { return data.profiles.find((item) => item.id === id)?.name || id }
function label(value: string): string { return value.split('-').map((part) => part.charAt(0).toUpperCase() + part.slice(1)).join(' ') }
function formatTime(value: string): string { const date = new Date(value); return Number.isNaN(date.getTime()) ? value : date.toLocaleString() }
function formatBytes(value: number): string { if (!value) return '0 B'; if (value < 1024) return `${value} B`; if (value < 1024 ** 2) return `${(value / 1024).toFixed(1)} KB`; if (value < 1024 ** 3) return `${(value / 1024 ** 2).toFixed(1)} MB`; return `${(value / 1024 ** 3).toFixed(1)} GB` }
function progressPercent(done: number, total: number): number { if (total <= 0) return done > 0 ? 100 : 0; return Math.max(0, Math.min(100, Math.round((done / total) * 100))) }
function errorText(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }
