import { useEffect, useMemo, useState } from 'react'
import { cancellationAvailability, lifecycleRecordFor } from '../lifecycle'
import { formatBytes, formatDateTime, lifecycleStateLabel, statusLabel } from '../i18n/format'
import {
  localRecoveryAPI,
  newRecoveryKey,
  type LocalRecoveryPreflight,
  type LocalRecoveryState,
  type RecoveryWorkspaceData,
} from '../localRecovery'
import { PortabilityWorkspace } from './PortabilityWorkspace'

export function RecoveryWorkspace({ data, onRefresh }: { data: RecoveryWorkspaceData; onRefresh: () => Promise<void> }) {
  const [state, setState] = useState<LocalRecoveryState>(() => localRecoveryAPI.emptyState())
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const nativeMode = localRecoveryAPI.isNative()

  const refresh = async () => {
    try { setState(await localRecoveryAPI.state()) }
    catch (reason) { setError(errorText(reason)) }
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
    } finally { setBusy('') }
  }

  const requirePreflight = async (profileId: string, allowed: keyof Pick<LocalRecoveryPreflight, 'snapshotAllowed' | 'archiveAllowed' | 'unarchiveAllowed' | 'trashAllowed'>) => {
    const result = await localRecoveryAPI.preflight(profileId)
    if (!result[allowed]) throw new Error(result.reasons?.join(' · ') || `本地恢复预检拒绝了操作：${allowed}`)
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
    if (!window.confirm('确定将此环境及其受管浏览器数据移入可恢复回收站并保留 30 天吗？')) throw new Error('已取消移入回收站。')
    return localRecoveryAPI.trash({ profileId, retentionDays: 30, idempotencyKey: newRecoveryKey() })
  })
  const restoreSnapshot = (snapshotId: string) => run(`restore-${snapshotId}`, async () => {
    const name = window.prompt('可选：为恢复后的新环境输入名称', '') || ''
    return localRecoveryAPI.restoreSnapshot({ snapshotId, name, idempotencyKey: newRecoveryKey() })
  })
  const restoreTrash = (profileId: string, trashId: string) => run(`restore-trash-${trashId}`, () => localRecoveryAPI.restoreTrash({ profileId, trashId, idempotencyKey: newRecoveryKey() }))
  const permanentDelete = (profileId: string, trashId: string) => run(`delete-${trashId}`, async () => {
    const confirmation = window.prompt(`永久删除无法撤销。请输入完整环境 ID：\n${profileId}`, '') || ''
    if (confirmation !== profileId) throw new Error('输入的环境 ID 与确认要求不一致。')
    return localRecoveryAPI.permanentDelete({ profileId, trashId, confirmation, idempotencyKey: newRecoveryKey() })
  })
  const cancel = (operationId: string) => run(`cancel-${operationId}`, () => localRecoveryAPI.cancel(operationId))
  const manualRefresh = () => run('refresh', () => localRecoveryAPI.refresh())

  return <>
    <div className="page-heading compact">
      <div><span className="eyebrow">本地数据保护</span><h1>数据与恢复</h1><p>创建经过验证的同机快照、恢复为新环境、归档环境并管理可恢复回收站。</p></div>
      <button className="button secondary" disabled={!nativeMode || Boolean(busy)} onClick={() => void manualRefresh()}>{busy === 'refresh' ? '正在检查…' : '刷新恢复状态'}</button>
    </div>
    {!nativeMode && <div className="form-error page-form-error">本地恢复操作需要 Wails 桌面运行时。</div>}
    {error && <div className="form-error page-form-error">{error}</div>}

    <section className="panel recovery-summary">
      <Summary label="已验证快照" value={state.snapshots.filter((item) => item.status === 'verified').length} detail={`共 ${state.snapshots.length} 条快照记录`} />
      <Summary label="可恢复回收站" value={state.trash.filter((item) => item.status === 'stored').length} detail={`共保留 ${state.trash.length} 条记录`} />
      <Summary label="进行中" value={state.progress.filter((item) => item.status === 'running' || item.status === 'pending').length} detail="受限的本地操作" />
      <Summary label="需要人工检查" value={state.trashReconciliation.findings?.length || 0} detail="系统不会猜测权威副本" warn={Boolean(state.trashReconciliation.findings?.length)} />
    </section>

    <section className="panel recovery-section">
      <div className="panel-heading"><div><h2>环境操作</h2><p>生命周期状态、活动会话、操作锁和存储预检会共同限制可用操作。</p></div></div>
      <div className="recovery-profile-list">
        {data.profiles.length === 0 ? <Empty text="当前没有可管理的浏览器环境。" /> : data.profiles.map((profile) => {
          const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
          const active = data.sessions.some((item) => item.profileId === profile.id && ['starting', 'ready', 'stopping'].includes(item.state))
          const locked = Boolean(record?.lock)
          const disabled = !nativeMode || active || locked || Boolean(busy)
          return <article className="recovery-row" key={profile.id}>
            <div className="recovery-identity"><div className="avatar">{profile.name.slice(0, 2).toUpperCase()}</div><div><strong>{profile.name}</strong><span>{profile.id}</span></div></div>
            <div className="recovery-state"><span className={`lifecycle-pill ${record?.state || 'missing'} ${locked ? 'locked' : ''}`}>{lifecycleStateLabel(record?.state, locked)}</span><small>{active ? '浏览器会话正在运行' : locked ? `操作锁：${record?.lock?.operationId}` : '没有活动阻止项'}</small></div>
            <div className="recovery-actions">
              {record?.state === 'available' && <button className="button secondary" disabled={disabled} onClick={() => void snapshot(profile.id)}>{busy === `snapshot-${profile.id}` ? '正在创建快照…' : '创建快照'}</button>}
              {(record?.state === 'available' || record?.state === 'draft') && <button className="button secondary" disabled={disabled} onClick={() => void archive(profile.id)}>{busy === `archive-${profile.id}` ? '正在归档…' : '归档'}</button>}
              {record?.state === 'archived' && <button className="button secondary" disabled={disabled} onClick={() => void unarchive(profile.id)}>{busy === `unarchive-${profile.id}` ? '正在恢复状态…' : '取消归档'}</button>}
              {record && ['available', 'draft', 'archived'].includes(record.state) && <button className="button secondary danger-text" disabled={disabled} onClick={() => void trash(profile.id)}>{busy === `trash-${profile.id}` ? '正在移动…' : '移入回收站'}</button>}
            </div>
          </article>
        })}
      </div>
    </section>

    <section className="panel recovery-section">
      <div className="panel-heading"><div><h2>本地快照</h2><p>快照是同一用户、同一设备的恢复数据，不会覆盖来源环境。</p></div></div>
      <div className="recovery-card-grid">
        {state.snapshots.length === 0 ? <Empty text="还没有创建本地快照。" /> : [...state.snapshots].reverse().map((item) => <article className="recovery-card" key={item.snapshotId}>
          <div className="recovery-card-head"><strong>{profileName(data, item.sourceProfileId)}</strong><span className={`lifecycle-operation-status ${item.status}`}>{recoveryStatusLabel(item.status)}</span></div>
          <code>{item.snapshotId}</code>
          <dl><div><dt>创建时间</dt><dd>{formatDateTime(item.createdAt)}</dd></div><div><dt>数据内容</dt><dd>{item.fileCount} 个文件 · {formatBytes(item.totalBytes)}</dd></div><div><dt>目录树身份</dt><dd>{item.treeDigest.slice(0, 16)}…</dd></div></dl>
          <button className="button primary" disabled={!nativeMode || item.status !== 'verified' || Boolean(busy)} onClick={() => void restoreSnapshot(item.snapshotId)}>{busy === `restore-${item.snapshotId}` ? '正在恢复…' : '恢复为新环境'}</button>
        </article>)}
      </div>
    </section>

    <section className="panel recovery-section">
      <div className="panel-heading"><div><h2>可恢复回收站</h2><p>保留期限只是可见元数据。永久删除始终要求输入完整环境 ID。</p></div></div>
      <div className="recovery-card-grid">
        {state.trash.filter((item) => item.status !== 'deleted').length === 0 ? <Empty text="可恢复回收站为空。" /> : state.trash.filter((item) => item.status !== 'deleted').map((item) => <article className="recovery-card" key={item.trashId}>
          <div className="recovery-card-head"><strong>{profileName(data, item.profileId)}</strong><span className={`lifecycle-operation-status ${item.status}`}>{recoveryStatusLabel(item.status)}</span></div>
          <code>{item.trashId}</code>
          <dl><div><dt>原始状态</dt><dd>{lifecycleStateLabel(item.originalState)}</dd></div><div><dt>数据内容</dt><dd>{item.fileCount} 个文件 · {formatBytes(item.totalBytes)}</dd></div><div><dt>保留期限</dt><dd>{formatDateTime(item.retentionDeadline)}</dd></div></dl>
          <div className="recovery-card-actions"><button className="button primary" disabled={!nativeMode || item.status !== 'stored' || !item.dataPresent || Boolean(busy)} onClick={() => void restoreTrash(item.profileId, item.trashId)}>{busy === `restore-trash-${item.trashId}` ? '正在恢复…' : '恢复环境'}</button><button className="button secondary danger-text" disabled={!nativeMode || item.status !== 'stored' || Boolean(busy)} onClick={() => void permanentDelete(item.profileId, item.trashId)}>{busy === `delete-${item.trashId}` ? '正在删除…' : '永久删除'}</button></div>
        </article>)}
      </div>
    </section>

    <section className="panel recovery-section">
      <div className="panel-heading"><div><h2>操作进度与历史</h2><p>只有到达声明的安全边界后，取消请求才会生效。</p></div></div>
      <div className="recovery-operation-list">
        {state.progress.length === 0 && operations.length === 0 ? <Empty text="还没有本地恢复操作记录。" /> : state.progress.map((item) => <article className="recovery-operation" key={item.operationId}>
          <div><strong>{operationLabel(item.operationType)}</strong><span>{item.operationId}</span></div>
          <div className="recovery-progress"><div><span style={{ width: `${progressPercent(item.bytesProcessed, item.bytesTotal)}%` }} /></div><small>{operationLabel(item.stage)} · {formatBytes(item.bytesProcessed)} / {formatBytes(item.bytesTotal)}</small></div>
          <span className={`lifecycle-operation-status ${item.status}`}>{recoveryStatusLabel(item.status)}</span>
          <button className="button secondary" disabled={!item.cancellationAvailable || Boolean(busy)} onClick={() => void cancel(item.operationId)}>安全取消</button>
        </article>)}
        {state.progress.length === 0 && operations.slice(0, 8).map((item) => <article className="recovery-operation" key={item.id}><div><strong>{operationLabel(item.type)}</strong><span>{item.id}</span></div><div><strong>{operationLabel(item.stage)}</strong><small>{cancellationText(item)}</small></div><span className={`lifecycle-operation-status ${item.status}`}>{recoveryStatusLabel(item.status)}</span></article>)}
      </div>
    </section>

    <section className="panel recovery-section">
      <div className="panel-heading"><div><h2>需要人工恢复的状态</h2><p>中断或互相矛盾的存储状态会被保留供人工检查，应用不会猜测哪一份数据才是权威副本。</p></div></div>
      {(state.trashReconciliation.findings?.length || 0) === 0 ? <Empty text="没有发现互相矛盾的回收站状态。" /> : <ul className="lifecycle-findings">{state.trashReconciliation.findings?.map((item) => <li className="warn" key={`${item.trashId}-${item.reasonCode}`}><strong>{item.reasonCode}</strong><span>{item.profileId} · 来源 {item.sourceState} · 回收站 {item.trashState} · 元数据 {item.profileState}</span></li>)}</ul>}
    </section>

    <PortabilityWorkspace data={data} onRefresh={onRefresh} />
  </>
}

function Summary({ label: title, value, detail, warn = false }: { label: string; value: number; detail: string; warn?: boolean }) {
  return <div className={`recovery-summary-item ${warn ? 'warn' : ''}`}><span>{title}</span><strong>{value}</strong><small>{detail}</small></div>
}
function Empty({ text }: { text: string }) { return <div className="lifecycle-empty">{text}</div> }
function profileName(data: RecoveryWorkspaceData, id: string): string { return data.profiles.find((item) => item.id === id)?.name || id }
function progressPercent(done: number, total: number): number { if (total <= 0) return done > 0 ? 100 : 0; return Math.max(0, Math.min(100, Math.round((done / total) * 100))) }
function errorText(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }

function operationLabel(value: string): string {
  const labels: Record<string, string> = { snapshot: '创建快照', restore: '恢复快照', archive: '归档', unarchive: '取消归档', trash: '移入回收站', 'restore-trash': '恢复回收站环境', 'permanent-delete': '永久删除', preflight: '预检', copying: '复制数据', verifying: '验证数据', publishing: '发布结果', completed: '完成' }
  return labels[value] || value.split('-').join(' ')
}
function recoveryStatusLabel(value: string): string {
  const labels: Record<string, string> = { verified: '已验证', stored: '已保存', deleted: '已删除', pending: '等待中', running: '进行中', completed: '已完成', partial: '部分完成', cancelled: '已取消', failed: '失败', 'recovery-required': '需要恢复', recovered: '已恢复' }
  return labels[value] || statusLabel(value)
}
function cancellationText(operation: Parameters<typeof cancellationAvailability>[0]): string {
  if (operation.cancellationRequested) return '已请求取消，将在安全边界生效'
  if (!['pending', 'running'].includes(operation.status)) return '操作已结束，不能取消'
  if (operation.safeCancellationStage) return `可在 ${operationLabel(operation.safeCancellationStage)} 安全取消`
  return '当前阶段暂不能取消'
}
