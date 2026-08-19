import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  cancellationAvailability,
  normalizeLifecycleBootstrap,
  type LifecycleBootstrap,
  type LifecycleOperation,
} from '../lifecycle'
import { ui } from '../i18n'
import { formatDateTime } from '../i18n/format'
import { backend } from '../lib/backend'
import { multiProfileAPI, type OperationReportExportResult } from '../multiProfile'
import { BulkLifecycleWorkspace } from './BulkLifecycleWorkspace'
import { MultiProfileWorkspace } from './MultiProfileWorkspace'
import { StorageLocationsWorkspace } from './StorageLocationsWorkspace'
import { TemplateMaintenanceWorkspace } from './TemplateMaintenanceWorkspace'

const emptyData: LifecycleBootstrap = normalizeLifecycleBootstrap({
  version: 'loading',
  profiles: [],
  providers: [],
  kernels: [],
  adapters: [],
  sessions: [],
  credentials: [],
  credentialProvider: 'Operating-system keyring',
  adapterPins: [],
  kernelPins: [],
  runtimePlatform: 'browser',
  runtimeArch: 'unknown',
})

const phase5OperationTypes = new Set([
  'export-definition',
  'import-definition',
  'create-template',
  'apply-template',
  'bulk-metadata-update',
  'bulk-health-refresh',
  'storage-reconcile',
  'archive',
  'unarchive',
  'trash',
])

type Phase5Section = 'operations' | 'lifecycle' | 'profiles' | 'storage' | 'templates'

type NativeLifecycleAPI = {
  CancelLocalRecoveryOperation(operationId: string): Promise<unknown>
}

function nativeLifecycleAPI(): NativeLifecycleAPI | undefined {
  return (window as unknown as { go?: { main?: { DesktopApp?: NativeLifecycleAPI } } }).go?.main?.DesktopApp
}

const sections: Array<{ key: Phase5Section; label: string; detail: string }> = [
  { key: 'operations', label: ui.batch.operations, detail: '查看进度、结果、脱敏报告和安全取消状态' },
  { key: 'lifecycle', label: ui.batch.lifecycle, detail: '受限的归档、取消归档和可恢复回收站操作' },
  { key: 'profiles', label: ui.batch.profiles, detail: '批量元数据、健康检查、便携导出和存储盘点' },
  { key: 'storage', label: ui.batch.locations, detail: '查看固定受管路径与系统卷信息' },
  { key: 'templates', label: ui.batch.templates, detail: '检查和维护不包含秘密的环境模板' },
]

export function MultiProfileToolsPage() {
  const [section, setSection] = useState<Phase5Section>('operations')
  const [data, setData] = useState<LifecycleBootstrap>(emptyData)
  const [loading, setLoading] = useState(false)
  const [cancelling, setCancelling] = useState('')
  const [exportingReport, setExportingReport] = useState('')
  const [reportResult, setReportResult] = useState<OperationReportExportResult>()
  const [error, setError] = useState('')
  const refreshing = useRef(false)

  const refresh = useCallback(async () => {
    if (refreshing.current) return
    refreshing.current = true
    setLoading(true)
    try {
      setData(normalizeLifecycleBootstrap(await backend.bootstrap()))
      setError('')
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      refreshing.current = false
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
    const timer = window.setInterval(() => void refresh(), 2500)
    return () => window.clearInterval(timer)
  }, [refresh])

  const phase5Operations = useMemo(() => [...data.lifecycleOperations]
    .filter((operation) => phase5OperationTypes.has(operation.type))
    .sort((left, right) => right.updatedAt.localeCompare(left.updatedAt)), [data.lifecycleOperations])
  const operations = phase5Operations.slice(0, 12)
  const runningCount = phase5Operations.filter((operation) => ['pending', 'running'].includes(operation.status)).length
  const recoveryCount = phase5Operations.filter((operation) => operation.status === 'recovery-required').length
  const reviewCount = phase5Operations.filter((operation) => ['partial', 'failed'].includes(operation.status)).length

  const cancelOperation = async (operation: LifecycleOperation) => {
    const api = nativeLifecycleAPI()
    if (!api) {
      setError('取消操作需要 Wails 桌面运行时。')
      return
    }
    setCancelling(operation.id)
    setError('')
    try {
      await api.CancelLocalRecoveryOperation(operation.id)
      await refresh()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setCancelling('')
    }
  }

  const exportOperationReport = async (operation: LifecycleOperation) => {
    if (!multiProfileAPI.isNative()) {
      setError('导出操作报告需要 Wails 桌面运行时。')
      return
    }
    setExportingReport(operation.id)
    setReportResult(undefined)
    setError('')
    try {
      const destinationPath = await multiProfileAPI.pickOperationReportFile(operation.id)
      if (!destinationPath) return
      const result = await multiProfileAPI.exportOperationReport({ operationId: operation.id, destinationPath })
      setReportResult(result)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setExportingReport('')
    }
  }

  return <div className="multi-profile-page">
    <div className="multi-profile-dock-header">
      <div><span className="eyebrow">{ui.batch.eyebrow}</span><h1>{ui.batch.title}</h1><p>{ui.batch.description}</p></div>
      <button className="button secondary" disabled={loading} onClick={() => void refresh()}>{loading ? ui.batch.refreshing : ui.batch.refreshData}</button>
    </div>
    {error && <div className="form-error page-form-error">{error}</div>}
    {reportResult && <div className="info-banner"><strong>操作报告已保存</strong><p>{reportResult.path}</p><code>{reportResult.payloadSha256}</code></div>}

    <nav className="batch-section-tabs" aria-label="批量管理功能">
      {sections.map((item) => <button
        key={item.key}
        className={section === item.key ? 'active' : ''}
        aria-pressed={section === item.key}
        title={item.detail}
        onClick={() => setSection(item.key)}
      >{item.label}</button>)}
    </nav>
    <p className="batch-section-detail">{sections.find((item) => item.key === section)?.detail}</p>

    {section === 'operations' && <>
      <section className="panel recovery-summary" aria-label="批量操作摘要">
        <Summary label="环境" value={data.profiles.length} detail="当前本机环境记录" />
        <Summary label="进行中" value={runningCount} detail="等待或正在执行的操作" warn={runningCount > 0} />
        <Summary label="需要检查" value={reviewCount} detail="部分完成或失败的操作" warn={reviewCount > 0} />
        <Summary label="需要恢复" value={recoveryCount} detail="已保留的模糊状态" warn={recoveryCount > 0} />
      </section>
      <Phase5OperationJournal
        data={data}
        operations={operations}
        cancelling={cancelling}
        exportingReport={exportingReport}
        onCancel={cancelOperation}
        onExportReport={exportOperationReport}
      />
    </>}
    {section === 'lifecycle' && <BulkLifecycleWorkspace data={data} onRefresh={refresh} />}
    {section === 'profiles' && <MultiProfileWorkspace data={data} onRefresh={refresh} />}
    {section === 'storage' && <StorageLocationsWorkspace />}
    {section === 'templates' && <TemplateMaintenanceWorkspace />}
  </div>
}

export function MultiProfileDock() {
  return null
}

function Summary({ label: title, value, detail, warn = false }: { label: string; value: number; detail: string; warn?: boolean }) {
  return <div className={`recovery-summary-item ${warn ? 'warn' : ''}`}><span>{title}</span><strong>{value}</strong><small>{detail}</small></div>
}

function Phase5OperationJournal({
  data,
  operations,
  cancelling,
  exportingReport,
  onCancel,
  onExportReport,
}: {
  data: LifecycleBootstrap
  operations: LifecycleOperation[]
  cancelling: string
  exportingReport: string
  onCancel: (operation: LifecycleOperation) => Promise<void>
  onExportReport: (operation: LifecycleOperation) => Promise<void>
}) {
  const nativeMode = multiProfileAPI.isNative()
  return <section className="panel recovery-section batch-operation-journal">
    <div className="panel-heading">
      <div><span className="eyebrow">权威生命周期日志</span><h2>批量操作记录</h2><p>状态、固定环境选择、逐项结果、安全取消和脱敏报告均来自现有操作日志。</p></div>
      <span className="lifecycle-operation-status running">最近 {operations.length} 条</span>
    </div>
    {operations.length === 0 ? <div className="lifecycle-empty">还没有便携导入导出、模板、批量或可恢复生命周期操作记录。</div> : <ul className="lifecycle-operations">
      {operations.map((operation) => {
        const cancellationAllowed = canCancel(operation)
        const summary = itemSummary(operation)
        return <li key={operation.id}>
          <div><strong>{operationLabel(operation.type)}</strong><span>{profileSelectionLabel(data, operation)} · 更新于 {formatDateTime(operation.updatedAt)}</span><code title={operation.id}>{operation.id}</code></div>
          <span className={`lifecycle-operation-status ${operation.status}`}>{operationStatusLabel(operation.status)}</span>
          <div className="lifecycle-operation-detail"><strong>{operationLabel(operation.stage)}</strong><span>{summary || cancellationText(operation)}</span></div>
          <div className="toolbar batch-operation-actions">
            <small>{cancellationText(operation)}</small>
            <button className="button secondary" disabled={!nativeMode || Boolean(cancelling) || Boolean(exportingReport)} onClick={() => void onExportReport(operation)}>{exportingReport === operation.id ? '正在导出报告…' : '导出脱敏报告'}</button>
            {(cancellationAllowed || operation.cancellationRequested) && <button className="button secondary" disabled={!cancellationAllowed || Boolean(cancelling) || Boolean(exportingReport)} onClick={() => void onCancel(operation)}>{operation.cancellationRequested ? '已请求取消' : cancelling === operation.id ? '正在请求…' : '安全取消'}</button>}
          </div>
        </li>
      })}
    </ul>}
  </section>
}

function canCancel(operation: LifecycleOperation): boolean {
  return ['pending', 'running'].includes(operation.status) && !operation.cancellationRequested && Boolean(operation.safeCancellationStage)
}

function itemSummary(operation: LifecycleOperation): string {
  const items = operation.items || []
  if (items.length === 0) return ''
  const counts = new Map<string, number>()
  for (const item of items) counts.set(item.status, (counts.get(item.status) || 0) + 1)
  return [...counts.entries()].map(([status, count]) => `${count} 个${operationStatusLabel(status)}`).join(' · ')
}

function profileSelectionLabel(data: LifecycleBootstrap, operation: LifecycleOperation): string {
  const names = operation.profileIds.map((profileId) => data.profiles.find((profile) => profile.id === profileId)?.name || profileId)
  if (names.length <= 3) return names.join('、') || '没有环境'
  return `${names.slice(0, 3).join('、')} 等 ${names.length} 个环境`
}

function operationLabel(value: string): string {
  const labels: Record<string, string> = {
    'export-definition': '导出环境定义',
    'import-definition': '导入环境定义',
    'create-template': '创建模板',
    'apply-template': '应用模板',
    'bulk-metadata-update': '批量更新元数据',
    'bulk-health-refresh': '批量健康检查',
    'storage-reconcile': '存储检查',
    archive: '归档环境',
    unarchive: '取消归档',
    trash: '移入回收站',
  }
  return labels[value] || value.split('-').join(' ')
}

function operationStatusLabel(value: string): string {
  const labels: Record<string, string> = {
    pending: '等待中', running: '进行中', completed: '已完成', partial: '部分完成', cancelled: '已取消', failed: '失败', 'recovery-required': '需要恢复', recovered: '已恢复', succeeded: '成功', skipped: '已跳过',
  }
  return labels[value] || value
}

function cancellationText(operation: LifecycleOperation): string {
  if (operation.cancellationRequested) return '已请求取消，将在安全边界生效'
  if (!['pending', 'running'].includes(operation.status)) return '操作已结束，不能取消'
  if (operation.safeCancellationStage) return `可在 ${operationLabel(operation.safeCancellationStage)} 安全取消`
  const fallback = cancellationAvailability(operation)
  return fallback ? '当前阶段暂不能取消' : '当前阶段暂不能取消'
}
