import { useEffect, useMemo, useState, type ChangeEvent } from 'react'
import { lifecycleRecordFor, type LifecycleBootstrap, type LifecycleState } from '../lifecycle'
import {
  multiProfileAPI,
  newMultiProfileKey,
  type BulkLifecycleAction,
  type BulkLifecycleResult,
} from '../multiProfile'

export function BulkLifecycleWorkspace({ data, onRefresh }: { data: LifecycleBootstrap; onRefresh: () => Promise<void> }) {
  const nativeMode = multiProfileAPI.isNative()
  const [selected, setSelected] = useState<string[]>([])
  const [action, setAction] = useState<BulkLifecycleAction>('archive')
  const [retentionDays, setRetentionDays] = useState(30)
  const [confirmation, setConfirmation] = useState('')
  const [result, setResult] = useState<BulkLifecycleResult>()
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const selectable = useMemo(() => data.profiles.filter((profile) => {
    const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
    const active = data.sessions.some((session) => session.profileId === profile.id && ['starting', 'ready', 'stopping'].includes(session.state))
    return Boolean(record && ['available', 'draft', 'archived'].includes(record.state) && !record.lock && !active)
  }), [data.profiles, data.lifecycleRecords, data.sessions])

  const ready = useMemo(() => selected.length > 0 && selected.every((profileId) => {
    const record = lifecycleRecordFor(data.lifecycleRecords, profileId)
    return Boolean(record && lifecycleStateAllowed(action, record.state) && !record.lock && !data.sessions.some((session) => session.profileId === profileId && ['starting', 'ready', 'stopping'].includes(session.state)))
  }), [action, selected, data.lifecycleRecords, data.sessions])

  const expectedConfirmation = `TRASH ${selected.length} PROFILES`

  useEffect(() => {
    const known = new Set(selectable.map((profile) => profile.id))
    setSelected((current) => current.filter((profileId) => known.has(profileId)))
  }, [selectable])

  useEffect(() => {
    setConfirmation('')
  }, [action, selected.length])

  useEffect(() => {
    setResult(undefined)
  }, [action])

  const toggle = (profileId: string) => {
    setSelected((current) => current.includes(profileId)
      ? current.filter((item) => item !== profileId)
      : [...current, profileId])
  }

  const apply = async () => {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      if (!ready) throw new Error(`Every selected Profile must be stopped, unlocked, and eligible for ${action}.`)
      if (action === 'trash' && confirmation !== expectedConfirmation) {
        throw new Error(`Type ${expectedConfirmation} exactly to confirm recoverable trash.`)
      }
      const response = await multiProfileAPI.applyLifecycle({
        profileIds: selected,
        action,
        retentionDays: action === 'trash' ? retentionDays : undefined,
        confirmation: action === 'trash' ? confirmation : undefined,
        idempotencyKey: newMultiProfileKey(),
      })
      setResult(response)
      const succeeded = response.items.filter((item) => item.status === 'succeeded').length
      const remaining = response.items.length - succeeded
      setNotice(`${label(action)} ${response.status}: ${succeeded} succeeded${remaining ? `, ${remaining} need review` : ''}.`)
      setSelected([])
      setConfirmation('')
      await onRefresh()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setBusy(false)
    }
  }

  return <section className="panel bulk-lifecycle-workspace">
    <div className="panel-heading">
      <div><span className="eyebrow">Recoverable lifecycle</span><h2>Bulk archive, unarchive, or trash</h2><p>Run one authoritative M5.1/M5.2 lifecycle operation per selected Profile. Bulk permanent deletion is never available.</p></div>
      <span className="lifecycle-operation-status running">{selected.length} selected</span>
    </div>
    {!nativeMode && <div className="form-error">Bulk lifecycle actions require the Wails desktop runtime.</div>}
    {error && <div className="form-error">{error}</div>}
    {notice && <div className="info-banner"><strong>Completed</strong><p>{notice}</p></div>}

    <div className="bulk-lifecycle-controls">
      <label>Action<select value={action} disabled={busy} onChange={(event: ChangeEvent<HTMLSelectElement>) => setAction(event.target.value as BulkLifecycleAction)}>
        <option value="archive">Archive selected Profiles</option>
        <option value="unarchive">Unarchive selected Profiles</option>
        <option value="trash">Move selected Profiles to recoverable trash</option>
      </select></label>
      {action === 'trash' && <label>Retention days<input type="number" min={0} max={365} value={retentionDays} disabled={busy} onChange={(event: ChangeEvent<HTMLInputElement>) => setRetentionDays(Math.max(0, Math.min(365, Number(event.target.value) || 0)))} /></label>}
      <div className="toolbar">
        <button className="button secondary" disabled={busy || selectable.length === 0} onClick={() => setSelected(selectable.filter((profile) => {
          const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
          return Boolean(record && lifecycleStateAllowed(action, record.state))
        }).map((profile) => profile.id))}>Select eligible for {action}</button>
        <button className="button secondary" disabled={busy || selected.length === 0} onClick={() => setSelected([])}>Clear</button>
      </div>
    </div>

    <div className="bulk-lifecycle-list">
      {selectable.length === 0 ? <div className="lifecycle-empty">No stopped and unlocked Profiles are available.</div> : selectable.map((profile) => {
        const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
        const allowed = Boolean(record && lifecycleStateAllowed(action, record.state))
        return <label className={`bulk-lifecycle-row ${allowed ? '' : 'disabled'}`} key={profile.id}>
          <input type="checkbox" checked={selected.includes(profile.id)} disabled={busy || !allowed} onChange={() => toggle(profile.id)} />
          <div><strong>{profile.name}</strong><span>{profile.group || 'No group'} · {(profile.tags || []).join(', ') || 'No tags'}</span></div>
          <span className={`lifecycle-pill ${record?.state || 'missing'}`}>{record?.state || 'missing'}</span>
        </label>
      })}
    </div>

    {action === 'trash' && <div className="bulk-trash-confirmation">
      <div className="form-error"><strong>Recoverable, not permanent</strong><p>Browser data moves into Veilium private trash. Restore remains available until an explicit single-Profile permanent deletion.</p></div>
      <label>Type <code>{expectedConfirmation}</code><input value={confirmation} disabled={busy || selected.length === 0} onChange={(event: ChangeEvent<HTMLInputElement>) => setConfirmation(event.target.value)} /></label>
    </div>}

    {!ready && selected.length > 0 && <p className="muted">The fixed selection contains a Profile that is not eligible for {action}. Change the action or selection.</p>}
    <button className={`button ${action === 'trash' ? 'danger' : 'primary'}`} disabled={!nativeMode || busy || !ready || (action === 'trash' && confirmation !== expectedConfirmation)} onClick={() => void apply()}>
      {busy ? 'Applying…' : `${label(action)} ${selected.length || ''} selected Profile${selected.length === 1 ? '' : 's'}`}
    </button>

    {result && <div className="bulk-lifecycle-results">
      {result.items.map((item) => <article className={item.status} key={`${result.requestId}-${item.profileId}`}>
        <div><strong>{profileName(data, item.profileId)}</strong><span>{item.lifecycleState || 'unknown'}</span></div>
        <span className={`lifecycle-operation-status ${item.status}`}>{item.status}</span>
        {item.reasonCode && <small>{item.reasonCode}</small>}
      </article>)}
      <ul className="plain-list">{result.limitations?.map((item) => <li key={item}>{item}</li>)}</ul>
    </div>}
  </section>
}

function lifecycleStateAllowed(action: BulkLifecycleAction, state: LifecycleState): boolean {
  switch (action) {
    case 'archive': return state === 'available' || state === 'draft'
    case 'unarchive': return state === 'archived'
    case 'trash': return state === 'available' || state === 'draft' || state === 'archived'
  }
  return false
}

function label(action: BulkLifecycleAction): string {
  switch (action) {
    case 'archive': return 'Archive'
    case 'unarchive': return 'Unarchive'
    case 'trash': return 'Move to trash'
  }
  return action
}

function profileName(data: LifecycleBootstrap, profileId: string): string {
  return data.profiles.find((profile) => profile.id === profileId)?.name || profileId
}
