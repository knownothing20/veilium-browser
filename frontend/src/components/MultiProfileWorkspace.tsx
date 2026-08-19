import { useEffect, useMemo, useState, type ChangeEvent } from 'react'
import { formatBytes, formatDateTime, lifecycleStateLabel } from '../i18n/format'
import { lifecycleRecordFor } from '../lifecycle'
import type { RecoveryWorkspaceData } from '../localRecovery'
import {
  multiProfileAPI,
  newMultiProfileKey,
  type BulkHealthRefreshResult,
  type BulkPortableExportResult,
  type ProfileHealthReport,
  type StorageManagementState,
  type StorageRepairPlan,
} from '../multiProfile'
import type { IdentityMode } from '../portableProfiles'

export function MultiProfileWorkspace({ data, onRefresh }: { data: RecoveryWorkspaceData; onRefresh: () => Promise<void> }) {
  const nativeMode = multiProfileAPI.isNative()
  const [selected, setSelected] = useState<string[]>([])
  const [setGroup, setSetGroup] = useState(false)
  const [group, setGroupValue] = useState('')
  const [addTags, setAddTags] = useState('')
  const [removeTags, setRemoveTags] = useState('')
  const [exportDirectory, setExportDirectory] = useState('')
  const [exportMode, setExportMode] = useState<IdentityMode>('new-identity')
  const [bulkExport, setBulkExport] = useState<BulkPortableExportResult>()
  const [health, setHealth] = useState<BulkHealthRefreshResult>()
  const [storage, setStorage] = useState<StorageManagementState>()
  const [repairPlans, setRepairPlans] = useState<StorageRepairPlan[]>([])
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const eligible = useMemo(() => data.profiles.filter((profile) => {
    const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
    const active = data.sessions.some((session) => session.profileId === profile.id && ['starting', 'ready', 'stopping'].includes(session.state))
    return Boolean(record && ['available', 'draft'].includes(record.state) && !record.lock && !active)
  }), [data.profiles, data.lifecycleRecords, data.sessions])

  const exportReady = useMemo(() => selected.length > 0 && selected.every((profileId) => {
    const record = lifecycleRecordFor(data.lifecycleRecords, profileId)
    return record?.state === 'available' && !record.lock
  }), [selected, data.lifecycleRecords])

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
    if (selected.length === 0) throw new Error('请至少选择一个符合条件的环境。')
    const added = splitTags(addTags)
    const removed = splitTags(removeTags)
    if (!setGroup && added.length === 0 && removed.length === 0) throw new Error('请选择要修改的分组或标签。')
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
    setNotice(`批量元数据操作状态：${operationStatusLabel(result.operation.status)}。${succeeded} 个已更新${skipped ? `，${skipped} 个未修改` : ''}。`)
    setAddTags('')
    setRemoveTags('')
    await onRefresh()
  })

  const chooseExportDirectory = () => run('pick-export-directory', async () => {
    const path = await multiProfileAPI.pickExportDirectory()
    if (!path) {
      setNotice('已取消选择导出文件夹。')
      return
    }
    setExportDirectory(path)
    setBulkExport(undefined)
    setNotice('已选择导出文件夹。现有文件不会被覆盖。')
  })

  const exportProfiles = () => run('bulk-export', async () => {
    if (!exportReady) throw new Error('批量导出要求选中的环境处于可用、已停止且未锁定状态。')
    if (!exportDirectory) throw new Error('请先选择导出文件夹。')
    const result = await multiProfileAPI.exportProfiles({
      profileIds: selected,
      destinationDirectory: exportDirectory,
      identityMode: exportMode,
      idempotencyKey: newMultiProfileKey(),
    })
    setBulkExport(result)
    const succeeded = result.operation.items?.filter((item) => item.status === 'succeeded').length || 0
    const remaining = (result.operation.items?.length || 0) - succeeded
    setNotice(`便携导出状态：${operationStatusLabel(result.operation.status)}。已写入 ${succeeded} 个文件${remaining ? `，${remaining} 个未写入` : ''}。`)
    await onRefresh()
  })

  const refreshHealth = () => run('health', async () => {
    if (selected.length === 0) throw new Error('请至少选择一个符合条件的环境。')
    const result = await multiProfileAPI.refreshHealth({
      profileIds: selected,
      idempotencyKey: newMultiProfileKey(),
    })
    setHealth(result)
    const ready = result.reports.filter((item) => item.status === 'ready').length
    const limited = result.reports.filter((item) => item.status === 'limited').length
    const blocked = result.reports.filter((item) => item.status === 'blocked').length
    setNotice(`健康检查状态：${operationStatusLabel(result.operation.status)}。${ready} 个可启动，${limited} 个受限，${blocked} 个被阻止。`)
    await onRefresh()
  })

  const refreshStorage = () => run('storage', async () => {
    const result = await multiProfileAPI.reviewStorage()
    setStorage(result.state)
    setRepairPlans(result.repairPlans)
    setNotice(`已检查 ${result.state.inventory.profiles.length} 条环境存储记录，并生成 ${result.repairPlans.length} 条人工检查建议。`)
  })

  return <section className="panel recovery-section">
    <div className="panel-heading"><div><h2>多环境与存储管理</h2><p>对固定环境选择执行受限修改，导出不含秘密的定义，刷新本机启动健康状态，并在不自动清理或修复的前提下检查受管存储。</p></div></div>
    {!nativeMode && <div className="form-error">多环境和存储管理操作需要 Wails 桌面运行时。</div>}
    {error && <div className="form-error">{error}</div>}
    {notice && <div className="info-banner"><strong>操作结果</strong><p>{notice}</p></div>}

    <div className="settings-grid">
      <article className="panel setting-card">
        <div className="panel-heading"><div><h2>固定环境选择</h2><p>只能选择没有活动浏览器会话、没有生命周期锁，并处于可用或待完善状态的环境。</p></div><span className="lifecycle-operation-status running">已选择 {selected.length} 个</span></div>
        <div className="toolbar">
          <button className="button secondary" disabled={Boolean(busy) || eligible.length === 0} onClick={() => setSelected(eligible.map((item) => item.id))}>选择全部符合条件的环境</button>
          <button className="button secondary" disabled={Boolean(busy) || selected.length === 0} onClick={() => setSelected([])}>清空选择</button>
        </div>
        <div className="recovery-profile-list">
          {data.profiles.length === 0 ? <div className="lifecycle-empty">当前没有浏览器环境。</div> : data.profiles.map((profile) => {
            const record = lifecycleRecordFor(data.lifecycleRecords, profile.id)
            const active = data.sessions.some((session) => session.profileId === profile.id && ['starting', 'ready', 'stopping'].includes(session.state))
            const allowed = Boolean(record && ['available', 'draft'].includes(record.state) && !record.lock && !active)
            return <label className="recovery-row" key={profile.id}>
              <input type="checkbox" checked={selected.includes(profile.id)} disabled={!allowed || Boolean(busy)} onChange={() => toggle(profile.id)} />
              <div className="recovery-identity"><div className="avatar">{profile.name.slice(0, 2).toUpperCase()}</div><div><strong>{profile.name}</strong><span>{profile.group || '无分组'} · {(profile.tags || []).join('、') || '无标签'}</span></div></div>
              <div className="recovery-state"><span className={`lifecycle-pill ${record?.state || 'missing'} ${record?.lock ? 'locked' : ''}`}>{lifecycleStateLabel(record?.state, Boolean(record?.lock))}</span><small>{active ? '浏览器正在运行' : record?.lock ? '存在生命周期操作锁' : allowed ? '可以选择' : '当前不符合条件'}</small></div>
            </label>
          })}
        </div>
      </article>

      <article className="panel setting-card">
        <div className="panel-heading"><div><h2>批量元数据</h2><p>替换分组并添加或移除受限标签，不修改浏览器数据、网络路由、指纹参数或凭据。</p></div></div>
        <label className="checkbox-line"><input type="checkbox" checked={setGroup} onChange={(event: ChangeEvent<HTMLInputElement>) => setSetGroup(event.target.checked)} /><span>替换选中环境的分组</span></label>
        <label>分组<input value={group} disabled={!setGroup} maxLength={128} onChange={(event: ChangeEvent<HTMLInputElement>) => setGroupValue(event.target.value)} placeholder="留空表示清除分组" /></label>
        <label>添加标签<input value={addTags} onChange={(event: ChangeEvent<HTMLInputElement>) => setAddTags(event.target.value)} placeholder="使用逗号分隔多个标签" /></label>
        <label>移除标签<input value={removeTags} onChange={(event: ChangeEvent<HTMLInputElement>) => setRemoveTags(event.target.value)} placeholder="使用逗号分隔多个标签" /></label>
        <button className="button primary" disabled={!nativeMode || selected.length === 0 || Boolean(busy)} onClick={() => void updateMetadata()}>{busy === 'metadata' ? '正在更新…' : '执行受限元数据更新'}</button>
      </article>

      <article className="panel setting-card bulk-export-card">
        <div className="panel-heading"><div><h2>批量便携导出</h2><p>为每个处于可用状态的环境写入一个严格且不含秘密的 JSON 定义。现有文件始终不会被覆盖。</p></div></div>
        <label>身份模式<select value={exportMode} disabled={Boolean(busy)} onChange={(event: ChangeEvent<HTMLSelectElement>) => setExportMode(event.target.value as IdentityMode)}>
          <option value="new-identity">创建新身份（推荐）</option>
          <option value="preserve-identity">保留身份（高级）</option>
        </select></label>
        {exportMode === 'preserve-identity' && <div className="form-error">保留的身份材料不能同时用于多个设备或多个环境。浏览器证据和信任状态不会被导出。</div>}
        <div className="bulk-export-folder">
          <span title={exportDirectory}>{exportDirectory || '尚未选择导出文件夹'}</span>
          <button className="button secondary" disabled={!nativeMode || Boolean(busy)} onClick={() => void chooseExportDirectory()}>{busy === 'pick-export-directory' ? '正在选择…' : '选择文件夹'}</button>
        </div>
        {!exportReady && selected.length > 0 && <p className="muted">待完善环境仍可用于元数据和健康检查，但便携导出要求生命周期状态为 <strong>可用</strong>。</p>}
        <button className="button primary" disabled={!nativeMode || !exportReady || !exportDirectory || Boolean(busy)} onClick={() => void exportProfiles()}>{busy === 'bulk-export' ? '正在导出…' : `导出 ${selected.length || ''} 个选中环境`}</button>
        {bulkExport && <div className="bulk-export-results">
          {bulkExport.exports.map((item) => <article key={item.profileId}><strong>{item.profileName}</strong><span>{item.path}</span><code>{item.payloadSha256}</code></article>)}
        </div>}
      </article>

      <article className="panel setting-card bulk-health-card">
        <div className="panel-heading"><div><h2>批量健康检查</h2><p>重新检查生命周期、受管内核完整性、路由依赖、指纹策略、身份一致性和受管浏览器数据边界。</p></div><button className="button secondary" disabled={!nativeMode || selected.length === 0 || Boolean(busy)} onClick={() => void refreshHealth()}>{busy === 'health' ? '正在检查…' : '检查选中环境'}</button></div>
        {!health ? <div className="lifecycle-empty">请选择已停止的环境并运行只读健康检查。</div> : <div className="bulk-health-list">
          {health.reports.map((report) => <HealthReport key={report.profileId} report={report} />)}
        </div>}
      </article>

      <article className="panel setting-card storage-management-card">
        <div className="panel-heading"><div><h2>受管存储盘点</h2><p>统计不透明的环境文件，并报告缺失、孤立、不安全或不完整条目。所有建议均需人工处理，不会自动修改数据。</p></div><button className="button secondary" disabled={!nativeMode || Boolean(busy)} onClick={() => void refreshStorage()}>{busy === 'storage' ? '正在扫描…' : '刷新存储盘点'}</button></div>
        {!storage ? <div className="lifecycle-empty">运行存储扫描以检查当前受管环境数据。</div> : <>
          <dl>
            <div><dt>环境存储</dt><dd>{storage.inventory.profiles.length} 条记录 · {formatBytes(storage.inventory.summary.bytes)}</dd></div>
            <div><dt>已验证快照</dt><dd>{storage.snapshotCount} 个 · {formatBytes(storage.snapshotBytes)}</dd></div>
            <div><dt>可恢复回收站</dt><dd>{storage.trashCount} 条 · {formatBytes(storage.trashBytes)}</dd></div>
            <div><dt>生命周期历史</dt><dd>{storage.operationCount} 条操作</dd></div>
            <div><dt>扫描状态</dt><dd>{storage.inventory.incomplete ? '不完整' : '完整'} · {formatDateTime(storage.generatedAt)}</dd></div>
          </dl>
          <div className="recovery-profile-list">
            {storage.inventory.profiles.map((item) => <article className="recovery-row" key={item.profileId}>
              <div className="recovery-identity"><div><strong>{profileName(data, item.profileId)}</strong><span>{item.managedDir}</span></div></div>
              <div className="recovery-state"><span className={`lifecycle-operation-status ${item.status}`}>{inventoryStatusLabel(item.status)}</span><small>{item.summary.files} 个文件 · {formatBytes(item.summary.bytes)}{item.reasonCode ? ` · ${item.reasonCode}` : ''}</small></div>
            </article>)}
          </div>
          {Boolean(storage.inventory.orphans?.length || storage.inventory.unsafe?.length) && <ul className="lifecycle-findings">
            {storage.inventory.orphans?.map((item) => <li className="warn" key={`orphan-${item.relativePath}`}><strong>可能的孤立条目</strong><span>{item.relativePath} · {item.reasonCode}</span></li>)}
            {storage.inventory.unsafe?.map((item) => <li className="danger" key={`unsafe-${item.relativePath}`}><strong>不安全条目</strong><span>{item.relativePath} · {item.reasonCode}</span></li>)}
          </ul>}
          <div className="storage-repair-plans">
            <div className="panel-heading"><div><h3>人工检查建议</h3><p>这些只是建议，Veilium 绝不会自动执行。</p></div><span>{repairPlans.length}</span></div>
            {repairPlans.length === 0 ? <div className="lifecycle-empty">当前不需要修复或人工检查建议。</div> : repairPlans.map((plan) => <article className={`storage-repair-plan ${plan.risk}`} key={plan.id}>
              <div><strong>{repairPlanLabel(plan.kind)}</strong><span>{plan.profileId ? profileName(data, plan.profileId) : plan.relativePath || '存储盘点'}</span></div>
              <p>{plan.description}</p>
              <small>{plan.reasonCode} · 仅限人工处理</small>
            </article>)}
          </div>
          <ul className="plain-list">{storage.limitations?.map((item) => <li key={item}>{item}</li>)}</ul>
        </>}
      </article>
    </div>
  </section>
}

function HealthReport({ report }: { report: ProfileHealthReport }) {
  return <article className={`bulk-health-report ${report.status}`}>
    <div className="bulk-health-report-header">
      <div><strong>{report.profileName}</strong><span>{lifecycleStateLabel(report.lifecycleState)} · {formatDateTime(report.refreshedAt)}</span></div>
      <span className={`bulk-health-status ${report.status}`}>{healthStatusLabel(report.status)}</span>
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
    case 'lifecycle': return '生命周期'
    case 'kernel': return '受管浏览器内核'
    case 'route': return '网络路由与凭据'
    case 'fingerprint': return '指纹策略'
    case 'consistency': return '身份一致性'
    case 'managed-data': return '受管浏览器数据'
    default: return id
  }
}

function repairPlanLabel(kind: string): string {
  switch (kind) {
    case 'review-snapshot-restore': return '检查快照恢复方案'
    case 'inspect-missing-profile-data': return '检查缺失的环境数据'
    case 'review-orphan-ownership': return '检查孤立条目归属'
    case 'manual-security-review': return '人工安全检查'
    case 'rerun-bounded-inventory': return '重新运行受限存储盘点'
    default: return kind
  }
}

function splitTags(value: string): string[] {
  return value.split(/[\n,]/).map((item) => item.trim()).filter(Boolean)
}

function profileName(data: RecoveryWorkspaceData, profileId: string): string {
  return data.profiles.find((item) => item.id === profileId)?.name || profileId
}

function operationStatusLabel(value: string): string {
  const labels: Record<string, string> = { pending: '等待中', running: '进行中', completed: '已完成', partial: '部分完成', cancelled: '已取消', failed: '失败', 'recovery-required': '需要恢复', recovered: '已恢复' }
  return labels[value] || value
}
function healthStatusLabel(value: string): string { const labels: Record<string, string> = { ready: '可启动', limited: '受限', blocked: '已阻止' }; return labels[value] || value }
function inventoryStatusLabel(value: string): string { const labels: Record<string, string> = { present: '存在', missing: '缺失', unsafe: '不安全', incomplete: '不完整' }; return labels[value] || value }
function errorText(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }
