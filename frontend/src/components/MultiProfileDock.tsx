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

type NativeLifecycleAPI = {
  CancelLocalRecoveryOperation(operationId: string): Promise<unknown>
}

function nativeLifecycleAPI(): NativeLifecycleAPI | undefined {
  return (window as unknown as { go?: { main?: { DesktopApp?: NativeLifecycleAPI } } }).go?.main?.DesktopApp
}

export function MultiProfileDock() {
  const [open, setOpen] = useState(false)
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

  const operations = useMemo(() => [...data.lifecycleOperations]
    .filter((operation) => phase5OperationTypes.has(operation.type))
    .sort((left, right) => right.updatedAt.localeCompare(left.updatedAt))
    .slice(0, 12), [data.lifecycleOperations])

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
    <button className="multi-profile-dock-button" onClick={() => setOpen((value) => !value)} aria-expanded={open}>
      {open ? 'Close Phase 5 tools' : 'Multi-Profile tools'}
    </button>
    {open && <aside className="multi-profile-dock" aria-label="Multi-Profile and storage tools">
      <div className="multi-profile-dock-header">
        <div><span className="eyebrow">Phase 5 workspace</span><h1>Multi-Profile tools</h1><p>Bounded metadata, recoverable lifecycle, portability, health, and storage review.</p></div>
        <button className="button secondary" disabled={loading} onClick={() => void refresh()}>{loading ? 'Refreshing…' : 'Refresh data'}</button>
      </div>
      {error && <div className="form-error">{error}</div>}
      {reportResult && <div className="info-banner"><strong>Operation report saved</strong><p>{reportResult.path}</p><code>{reportResult.payloadSha256}</code></div>}
      <Phase5OperationJournal
        data={data}
        operations={operations}
        cancelling={cancelling}
        exportingReport={exportingReport}
        onCancel={cancelOperation}
        onExportReport={exportOperationReport}
      />
      <BulkLifecycleWorkspace data={data} onRefresh={refresh} />
      <MultiProfileWorkspace data={data} onRefresh={refresh} />
      <StorageLocationsWorkspace />
      <TemplateMaintenanceWorkspace />
    </aside>}
  </>
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
