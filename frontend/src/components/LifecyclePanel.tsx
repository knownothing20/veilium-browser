import type {
  LifecycleOperation,
  LifecycleRecord,
  LifecycleReconciliationReport,
} from '../lifecycle'

export function LifecyclePanel({
  records,
  operations,
  reconciliation,
}: {
  records: LifecycleRecord[]
  operations: LifecycleOperation[]
  reconciliation: LifecycleReconciliationReport
}) {
  const inventory = reconciliation.inventory
  const limited = records.filter((record) => record.state !== 'available').length
  const locked = records.filter((record) => Boolean(record.lock)).length
  const recovery = operations.filter((operation) => operation.status === 'recovery-required').length
  const missing = inventory.profiles.filter((profile) => profile.status === 'missing').length
  const unsafe = inventory.profiles.filter((profile) => profile.status === 'unsafe').length + (inventory.unsafe?.length || 0)
  const findings = [
    ...(reconciliation.actions || []).map((item) => ({
      key: `action-${item.kind}-${item.profileId || item.operationId || item.relativePath || item.reasonCode}`,
      title: label(item.kind),
      detail: [item.profileId, item.operationId, item.relativePath, item.reasonCode].filter(Boolean).join(' · '),
      tone: item.kind.includes('unsafe') || item.kind.includes('recovery') ? 'warn' : 'neutral',
    })),
    ...(inventory.orphans || []).map((item) => ({
      key: `orphan-${item.relativePath}`,
      title: 'Orphaned managed directory',
      detail: `${item.relativePath} · ${item.reasonCode}`,
      tone: 'warn',
    })),
    ...(inventory.unsafe || []).map((item) => ({
      key: `unsafe-${item.relativePath}`,
      title: 'Unsafe storage entry',
      detail: `${item.relativePath} · ${item.reasonCode}`,
      tone: 'danger',
    })),
  ].slice(0, 5)

  return (
    <section className="panel lifecycle-panel">
      <div className="panel-heading">
        <div>
          <h2>Profile lifecycle</h2>
          <p>Read-only M5.1 status. No archive, trash, restore or permanent-delete actions are enabled.</p>
        </div>
        <span className={`lifecycle-overall ${limited || locked || recovery || missing || unsafe ? 'attention' : 'clear'}`}>
          {limited || locked || recovery || missing || unsafe ? 'Attention required' : 'No lifecycle blockers'}
        </span>
      </div>
      <div className="lifecycle-summary">
        <Summary label="Available" value={records.length - limited} detail={`${records.length} lifecycle records`} />
        <Summary label="Limited" value={limited} detail="Draft, archived, trashed or invalid" tone={limited ? 'warn' : 'neutral'} />
        <Summary label="Locked" value={locked} detail="Conflicting operation protection" tone={locked ? 'warn' : 'neutral'} />
        <Summary label="Recovery" value={recovery} detail={`${missing} missing · ${unsafe} unsafe`} tone={recovery || missing || unsafe ? 'warn' : 'neutral'} />
      </div>
      <div className="lifecycle-storage-line">
        <span>Storage inventory</span>
        <strong>{formatBytes(inventory.summary.bytes)} · {inventory.summary.files} files</strong>
        <small>{inventory.incomplete ? 'Bounded scan incomplete' : `Generated ${formatTime(inventory.generatedAt)}`}</small>
      </div>
      {findings.length > 0 ? (
        <ul className="lifecycle-findings">
          {findings.map((item) => <li className={item.tone} key={item.key}><strong>{item.title}</strong><span>{item.detail}</span></li>)}
        </ul>
      ) : (
        <div className="lifecycle-empty">No startup recovery action, orphan, unsafe entry or missing managed directory was reported.</div>
      )}
    </section>
  )
}

function Summary({ label, value, detail, tone = 'neutral' }: { label: string; value: number; detail: string; tone?: string }) {
  return <div className={`lifecycle-summary-item ${tone}`}><span>{label}</span><strong>{value}</strong><small>{detail}</small></div>
}

function label(value: string): string {
  return value.split('-').map((part) => part.charAt(0).toUpperCase() + part.slice(1)).join(' ')
}

function formatBytes(value: number): string {
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  if (value < 1024 * 1024 * 1024) return `${(value / 1024 / 1024).toFixed(1)} MB`
  return `${(value / 1024 / 1024 / 1024).toFixed(1)} GB`
}

function formatTime(value: string): string {
  if (!value) return 'not scanned in browser preview'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
