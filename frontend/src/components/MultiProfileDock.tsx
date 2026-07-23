import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  cancellationAvailability,
  normalizeLifecycleBootstrap,
  type LifecycleBootstrap,
  type LifecycleOperation,
} from '../lifecycle'
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
  { key: 'operations', label: 'Operations', detail: 'History, progress, reports, and safe cancellation' },
  { key: 'lifecycle', label: 'Lifecycle', detail: 'Bounded archive, unarchive, and recoverable trash' },
  { key: 'profiles', label: 'Profiles', detail: 'Metadata, portable export, health, and storage inventory' },
  { key: 'storage', label: 'Locations', detail: 'Fixed managed paths and system-volume visibility' },
  { key: 'templates', label: 'Templates', detail: 'Inspect and maintain reusable non-secret defaults' },
]

export function MultiProfileDock() {
  const [open, setOpen] = useState(false)
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
    if (!open) return
    void refresh()
    const timer = window.setInterval(() => void refresh(), 2000)
    return () => window.clearInterval(timer)
  }, [open, refresh])

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
      setError('Operation cancellation requires the Wails desktop runtime.')
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
      setError('Operation report export requires the Wails desktop runtime.')
      return
    }
    setExportingReport(operation.id)
    setReportResult(undefined)
    setError('')
    try {
      const destinationPath = await multiProfileAPI.pickOperationReportFile(operation.id)
      if (!destinationPath) return
      const result = await multiProfileAPI.exportOperationReport({
        operationId: operation.id,
        destinationPath,
      })
      setReportResult(result)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setExportingReport('')
    }
  }

  return <>
    <button className="multi-profile-dock-button" onClick={() => setOpen((value) => !value)} aria-expanded={open} aria-controls="phase5-tools-workspace">
      {open ? 'Close Phase 5 tools' : 'Multi-Profile tools'}
    </button>
    {open && <aside id="phase5-tools-workspace" className="multi-profile-dock" aria-label="Multi-Profile and storage tools">
      <div className="multi-profile-dock-header">
        <div><span className="eyebrow">Phase 5 workspace</span><h1>Multi-Profile tools</h1><p>One bounded desktop path for portability, recoverable lifecycle, health, templates, operation history, and managed storage.</p></div>
        <button className="button secondary" disabled={loading} onClick={() => void refresh()}>{loading ? 'Refreshing…' : 'Refresh data'}</button>
      </div>
      {error && <div className="form-error">{error}</div>}
      {reportResult && <div className="info-banner"><strong>Operation report saved</strong><p>{reportResult.path}</p><code>{reportResult.payloadSha256}</code></div>}

      <nav className="toolbar" aria-label="Phase 5 tool sections">
        {sections.map((item) => <button
          key={item.key}
          className={`button ${section === item.key ? 'primary' : 'secondary'}`}
          aria-pressed={section === item.key}
          title={item.detail}
          onClick={() => setSection(item.key)}
        >
          {item.label}
        </button>)}
      </nav>
      <p className="muted">{sections.find((item) => item.key === section)?.detail}</p>

      {section === 'operations' && <>
        <section className="panel recovery-summary" aria-label="Phase 5 operation summary">
          <Summary label="Profiles" value={data.profiles.length} detail="Current local records" />
          <Summary label="Running" value={runningCount} detail="Pending or active operations" warn={runningCount > 0} />
          <Summary label="Needs review" value={reviewCount} detail="Partial or failed operations" warn={reviewCount > 0} />
          <Summary label="Recovery required" value={recoveryCount} detail="Preserved ambiguous state" warn={recoveryCount > 0} />
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
    </aside>}
  </>
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
  return <section className="panel recovery-section">
    <div className="panel-heading">
      <div><span className="eyebrow">Authoritative M5.1 journal</span><h2>Phase 5 operation history</h2><p>Live status, fixed Profile selection, per-item outcome, safe cancellation, recovery state, and redacted local report export from the existing lifecycle journal.</p></div>
      <span className="lifecycle-operation-status running">{operations.length} recent</span>
    </div>
    {operations.length === 0 ? <div className="lifecycle-empty">No portability, template, bulk, or recoverable lifecycle operation has been recorded.</div> : <ul className="lifecycle-operations">
      {operations.map((operation) => {
        const cancellationAllowed = canCancel(operation)
        const summary = itemSummary(operation)
        return <li key={operation.id}>
          <div>
            <strong>{label(operation.type)}</strong>
            <span>{profileSelectionLabel(data, operation)} · updated {formatTime(operation.updatedAt)}</span>
            <code title={operation.id}>{operation.id}</code>
          </div>
          <span className={`lifecycle-operation-status ${operation.status}`}>{operation.status}</span>
          <div className="lifecycle-operation-detail">
            <strong>{label(operation.stage)}</strong>
            <span>{summary || cancellationAvailability(operation)}</span>
          </div>
          <div className="toolbar">
            <small>{cancellationAvailability(operation)}</small>
            <button
              className="button secondary"
              disabled={!nativeMode || Boolean(cancelling) || Boolean(exportingReport)}
              onClick={() => void onExportReport(operation)}
            >
              {exportingReport === operation.id ? 'Exporting report…' : 'Export redacted report'}
            </button>
            {(cancellationAllowed || operation.cancellationRequested) && <button
              className="button secondary"
              disabled={!cancellationAllowed || Boolean(cancelling) || Boolean(exportingReport)}
              onClick={() => void onCancel(operation)}
            >
              {operation.cancellationRequested
                ? 'Cancellation requested'
                : cancelling === operation.id
                  ? 'Requesting…'
                  : 'Cancel safely'}
            </button>}
          </div>
        </li>
      })}
    </ul>}
  </section>
}

function canCancel(operation: LifecycleOperation): boolean {
  return ['pending', 'running'].includes(operation.status)
    && !operation.cancellationRequested
    && Boolean(operation.safeCancellationStage)
}

function itemSummary(operation: LifecycleOperation): string {
  const items = operation.items || []
  if (items.length === 0) return ''
  const counts = new Map<string, number>()
  for (const item of items) counts.set(item.status, (counts.get(item.status) || 0) + 1)
  return [...counts.entries()].map(([status, count]) => `${count} ${label(status).toLowerCase()}`).join(' · ')
}

function profileSelectionLabel(data: LifecycleBootstrap, operation: LifecycleOperation): string {
  const names = operation.profileIds.map((profileId) => data.profiles.find((profile) => profile.id === profileId)?.name || profileId)
  if (names.length <= 3) return names.join(', ')
  return `${names.slice(0, 3).join(', ')} +${names.length - 3} more`
}

function label(value: string): string {
  return value.split('-').map((part) => part.charAt(0).toUpperCase() + part.slice(1)).join(' ')
}

function formatTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
