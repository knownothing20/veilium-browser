import type { Bootstrap } from './types'

export type LifecycleState = 'available' | 'draft' | 'archived' | 'trashed' | 'invalid'
export type InventoryStatus = 'present' | 'missing' | 'unsafe' | 'incomplete'
export type LifecycleOperationStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'partial'
  | 'cancelled'
  | 'failed'
  | 'recovery-required'
  | 'recovered'

export interface LifecycleOperationLock {
  operationId: string
  acquiredAt: string
}

export interface LifecycleRecord {
  schemaVersion: number
  profileId: string
  state: LifecycleState
  managedDir: string
  createdAt: string
  updatedAt: string
  archivedAt?: string
  trashedAt?: string
  retentionDeadline?: string
  sourceId?: string
  lock?: LifecycleOperationLock
  recoveryCodes?: string[]
  limitationCodes?: string[]
  revision: number
}

export interface LifecycleOperationItem {
  itemId: string
  status: string
  startedAt?: string
  completedAt?: string
  completedStage?: string
  filesProcessed?: number
  bytesProcessed?: number
  reasonCode?: string
  outputId?: string
  recoveryId?: string
  limitations?: string[]
}

export interface LifecycleOperation {
  schemaVersion: number
  id: string
  type: string
  profileIds: string[]
  idempotencyKey?: string
  predecessorId?: string
  status: LifecycleOperationStatus
  stage: string
  startedAt: string
  updatedAt: string
  completedAt?: string
  cancellationRequested: boolean
  safeCancellationStage?: string
  items?: LifecycleOperationItem[]
  limitations?: string[]
  recoveryActions?: string[]
  stagingRef?: string
  quarantineRef?: string
  applicationVersion: string
  platform: string
  revision: number
}

export interface StorageSummary {
  files: number
  bytes: number
}

export interface ProfileStorage {
  profileId: string
  managedDir: string
  status: InventoryStatus
  summary: StorageSummary
  reasonCode?: string
  limitations?: string[]
}

export interface InventoryFinding {
  relativePath: string
  kind: string
  reasonCode: string
}

export interface StorageInventory {
  generatedAt: string
  managedRoot: string
  profiles: ProfileStorage[]
  orphans?: InventoryFinding[]
  unsafe?: InventoryFinding[]
  summary: StorageSummary
  incomplete: boolean
  limitations?: string[]
}

export interface ReconciliationAction {
  kind: string
  profileId?: string
  operationId?: string
  relativePath?: string
  reasonCode: string
}

export interface LifecycleReconciliationReport {
  generatedAt: string
  compatibilityCreated?: string[]
  actions?: ReconciliationAction[]
  inventory: StorageInventory
  limitations?: string[]
}

export interface LifecycleBootstrap extends Bootstrap {
  lifecycleRecords: LifecycleRecord[]
  lifecycleOperations: LifecycleOperation[]
  lifecycleReconciliation: LifecycleReconciliationReport
}

const emptyInventory: StorageInventory = {
  generatedAt: '',
  managedRoot: '.',
  profiles: [],
  orphans: [],
  unsafe: [],
  summary: { files: 0, bytes: 0 },
  incomplete: false,
  limitations: [],
}

export function normalizeLifecycleBootstrap(input: Bootstrap): LifecycleBootstrap {
  const lifecycle = input as Partial<LifecycleBootstrap>
  return {
    ...input,
    lifecycleRecords: lifecycle.lifecycleRecords || [],
    lifecycleOperations: lifecycle.lifecycleOperations || [],
    lifecycleReconciliation: lifecycle.lifecycleReconciliation || {
      generatedAt: '',
      compatibilityCreated: [],
      actions: [],
      inventory: emptyInventory,
      limitations: [],
    },
  }
}

export function lifecycleRecordFor(records: LifecycleRecord[], profileId: string): LifecycleRecord | undefined {
  return records.find((record) => record.profileId === profileId)
}

export function lifecycleAllowsLaunch(record: LifecycleRecord | undefined): boolean {
  return record?.state === 'available' && !record.lock
}

export function lifecycleAllowsEdit(record: LifecycleRecord | undefined): boolean {
  return Boolean(record && !record.lock && record.state !== 'archived' && record.state !== 'trashed')
}

export function lifecycleLabel(record: LifecycleRecord | undefined): string {
  if (!record) return 'missing lifecycle'
  if (record.lock) return `${record.state} · locked`
  return record.state
}
