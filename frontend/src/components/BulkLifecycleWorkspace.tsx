import { useEffect, useMemo, useState, type ChangeEvent } from 'react'
import { lifecycleRecordFor, type LifecycleBootstrap, type LifecycleState } from '../lifecycle'
import { lifecycleStateLabel } from '../i18n/format'
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
  useEffect(() => { setConfirmation('') }, [action, selected.length])
  useEffect(() => { setResult(undefined) }, [action])

  const toggle = (profileId: string) => setSelected((current) => current.includes(profileId) ? current.filter((item) => item !== profileId) : [...current, profileId])

  const apply = async () => {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      if (!ready) throw new Error(`每个选中的环境都必须已停止、未锁定并且允许执行“${actionLabel(action)}”。`)
      if (action === 'trash' && confirmation !== expectedConfirmation) throw new Error(`请完整输入 ${expectedConfirmation} 以确认移入可恢复回收站。`)
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
      setNotice(`${actionLabel(action)}：${succeeded} 个成功${remaining ? `，${remaining} 个需要检查` : ''}。`)
      setSelected([])
      setConfirmation('')
      await onRefresh()
    } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { setBusy(false) }
  }

  return <section className="panel bulk-lifecycle-workspace">
    <div className="panel-heading">
      <div><span className="eyebrow">可恢复生命周期操作</span><h2>批量归档、取消归档或移入回收站</h2><p>每个环境仍使用现有权威生命周期操作。批量永久删除始终不可用。</p></div>
      <span className="lifecycle-operation-status running">已选择 {selected.length} 个</span>
    </div>
    {!nativeMode && <div className="form-error">批量生命周期操作需要 Wails 桌面运行时。</div>}
    {error && <div className="form-error">{error}</div>}
    {notice && <div className="info-banner"><strong>操作完成</strong><p>{notice}</p></div>}

    <div className="bulk-lifecycle-controls">
      <label>操作<select value={action} disabled={busy} onChange={(event: ChangeEvent<HTMLSelectElement>) => setAction(event.target.value as BulkLifecycleAction)}>
        <option value="archive">归档选中的环境</option>
        <option value="unarchive">取消归档选中的环境</option>
        <option value="trash">将选中的环境移入可恢复回收站</option>
      </select></label>
      {action === 'trash' && <label>保留天数<input type="number" min={0} max={365} value={retentionDays} disabled={busy} onChange={(event: ChangeEvent<HTMLInputElement>) => setRetentionDays(Math.max(0, Math.min(365, Number(event.target.value) || 0)))} /></label>}
      <div className="toolbar">
        <button className="button secondary" disabled={busy || selectable.length === 0} onClick={() => setSelected(selectable.filter((profile) => {
          const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
          return Boolean(record && lifecycleStateAllowed(action, record.state))
        }).map((profile) => profile.id))}>选择所有符合条件的环境</button>
        <button className="button secondary" disabled={busy || selected.length === 0} onClick={() => setSelected([])}>清空选择</button>
      </div>
    </div>

    <div className="bulk-lifecycle-list">
      {selectable.length === 0 ? <div className="lifecycle-empty">没有已停止且未锁定的环境可供选择。</div> : selectable.map((profile) => {
        const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
        const allowed = Boolean(record && lifecycleStateAllowed(action, record.state))
        return <label className={`bulk-lifecycle-row ${allowed ? '' : 'disabled'}`} key={profile.id}>
          <input type="checkbox" checked={selected.includes(profile.id)} disabled={busy || !allowed} onChange={() => toggle(profile.id)} />
          <div><strong>{profile.name}</strong><span>{profile.group || '无分组'} · {(profile.tags || []).join('、') || '无标签'}</span></div>
          <span className={`lifecycle-pill ${record?.state || 'missing'}`}>{lifecycleStateLabel(record?.state)}</span>
        </label>
      })}
    </div>

    {action === 'trash' && <div className="bulk-trash-confirmation">
      <div className="form-error"><strong>这是可恢复操作，不是永久删除</strong><p>浏览器数据会移动到 Veilium 私有回收站。只有单个环境经过精确确认后才能永久删除。</p></div>
      <label>请完整输入 <code>{expectedConfirmation}</code><input value={confirmation} disabled={busy || selected.length === 0} onChange={(event: ChangeEvent<HTMLInputElement>) => setConfirmation(event.target.value)} /></label>
    </div>}

    {!ready && selected.length > 0 && <p className="muted">固定选择中包含不允许执行“{actionLabel(action)}”的环境，请修改操作或选择。</p>}
    <button className={`button ${action === 'trash' ? 'danger' : 'primary'}`} disabled={!nativeMode || busy || !ready || (action === 'trash' && confirmation !== expectedConfirmation)} onClick={() => void apply()}>{busy ? '正在执行…' : `${actionLabel(action)} ${selected.length || ''} 个环境`}</button>

    {result && <div className="bulk-lifecycle-results">
      {result.items.map((item) => <article className={item.status} key={`${result.requestId}-${item.profileId}`}>
        <div><strong>{profileName(data, item.profileId)}</strong><span>{lifecycleStateLabel(item.lifecycleState)}</span></div>
        <span className={`lifecycle-operation-status ${item.status}`}>{itemStatusLabel(item.status)}</span>
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
function actionLabel(action: BulkLifecycleAction): string { if (action === 'archive') return '归档'; if (action === 'unarchive') return '取消归档'; if (action === 'trash') return '移入回收站'; return action }
function itemStatusLabel(status: string): string { const labels: Record<string, string> = { succeeded: '成功', skipped: '已跳过', cancelled: '已取消', failed: '失败', pending: '等待中', running: '进行中' }; return labels[status] || status }
function profileName(data: LifecycleBootstrap, profileId: string): string { return data.profiles.find((profile) => profile.id === profileId)?.name || profileId }
