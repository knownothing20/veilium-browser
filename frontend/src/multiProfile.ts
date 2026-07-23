import type { LifecycleOperation, LifecycleOperationStatus, LifecycleState, StorageInventory } from './lifecycle'
import { normalizeLifecycleOperation } from './lifecycle'
import type { IdentityMode, PortableExportResult } from './portableProfiles'
import type { Profile } from './types'

function arrayOrEmpty<T>(value: T[] | null | undefined): T[] {
  return value || []
}

export interface BulkMetadataUpdateRequest {
  profileIds: string[]
  setGroup: boolean
  group?: string
  addTags?: string[]
  removeTags?: string[]
  idempotencyKey?: string
}

export interface BulkMetadataUpdateResult {
  operation: LifecycleOperation
  profiles: Profile[]
}

export type HealthCheckStatus = 'pass' | 'warning' | 'fail'
export type ProfileHealthStatus = 'ready' | 'limited' | 'blocked'

export interface BulkHealthRefreshRequest {
  profileIds: string[]
  idempotencyKey?: string
}

export interface ProfileHealthCheck {
  id: string
  status: HealthCheckStatus
  message: string
}

export interface ProfileHealthReport {
  profileId: string
  profileName: string
  lifecycleState: LifecycleState
  status: ProfileHealthStatus
  checks: ProfileHealthCheck[]
  refreshedAt: string
}

export interface BulkHealthRefreshResult {
  operation: LifecycleOperation
  reports: ProfileHealthReport[]
}

export type BulkLifecycleAction = 'archive' | 'unarchive' | 'trash'
export type BulkLifecycleItemStatus = 'succeeded' | 'skipped' | 'cancelled' | 'failed' | 'rolled-back' | 'recovery-required'

export interface BulkLifecycleRequest {
  profileIds: string[]
  action: BulkLifecycleAction
  retentionDays?: number
  confirmation?: string
  idempotencyKey: string
}

export interface BulkLifecycleItemResult {
  profileId: string
  status: BulkLifecycleItemStatus
  operationId?: string
  lifecycleState?: LifecycleState
  reasonCode?: string
  limitations?: string[]
}

export interface BulkLifecycleResult {
  requestId: string
  action: BulkLifecycleAction
  status: LifecycleOperationStatus
  items: BulkLifecycleItemResult[]
  completedAt: string
  limitations?: string[]
}

export interface BulkPortableExportRequest {
  profileIds: string[]
  destinationDirectory: string
  identityMode: IdentityMode
  idempotencyKey?: string
}

export interface BulkPortableExportResult {
  operation: LifecycleOperation
  destinationDirectory: string
  exports: PortableExportResult[]
}

export interface StorageManagementState {
  inventory: StorageInventory
  snapshotCount: number
  snapshotBytes: number
  trashCount: number
  trashBytes: number
  operationCount: number
  generatedAt: string
  limitations?: string[]
}

export interface StorageRepairPlan {
  id: string
  kind: string
  profileId?: string
  relativePath?: string
  reasonCode: string
  risk: 'danger' | 'warning' | 'info'
  description: string
  automatic: boolean
}

export interface StorageManagementReview {
  state: StorageManagementState
  repairPlans: StorageRepairPlan[]
}

export interface OperationReportExportRequest {
  operationId: string
  destinationPath: string
}

export interface OperationReportExportResult {
  operationId: string
  path: string
  payloadSha256: string
  generatedAt: string
  status: LifecycleOperationStatus
}

type NativeMultiProfileAPI = {
  BulkUpdateProfileMetadata(request: BulkMetadataUpdateRequest): Promise<BulkMetadataUpdateResult>
  BulkRefreshProfileHealth(request: BulkHealthRefreshRequest): Promise<BulkHealthRefreshResult>
  BulkApplyProfileLifecycle(request: BulkLifecycleRequest): Promise<BulkLifecycleResult>
  PickPortableExportDirectory(): Promise<string>
  BulkExportPortableProfiles(request: BulkPortableExportRequest): Promise<BulkPortableExportResult>
  RefreshStorageManagement(): Promise<StorageManagementState>
  ReviewStorageManagement(): Promise<StorageManagementReview>
  PickOperationReportFile(operationId: string): Promise<string>
  ExportLifecycleOperationReport(request: OperationReportExportRequest): Promise<OperationReportExportResult>
}

function native(): NativeMultiProfileAPI | undefined {
  return (window as unknown as { go?: { main?: { DesktopApp?: NativeMultiProfileAPI } } }).go?.main?.DesktopApp
}

function requireNative(): NativeMultiProfileAPI {
  const api = native()
  if (!api) throw new Error('Multi-Profile and storage actions require the Wails desktop runtime.')
  return api
}

export function newMultiProfileKey(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return `bulk-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function normalizeBulkMetadataUpdateResult(raw: BulkMetadataUpdateResult): BulkMetadataUpdateResult {
  return { ...raw, operation: normalizeLifecycleOperation(raw.operation), profiles: arrayOrEmpty(raw.profiles) }
}

function normalizeBulkHealthRefreshResult(raw: BulkHealthRefreshResult): BulkHealthRefreshResult {
  return { ...raw, operation: normalizeLifecycleOperation(raw.operation), reports: arrayOrEmpty(raw.reports) }
}

function normalizeBulkLifecycleResult(raw: BulkLifecycleResult): BulkLifecycleResult {
  return { ...raw, items: arrayOrEmpty(raw.items), limitations: arrayOrEmpty(raw.limitations) }
}

function normalizeBulkPortableExportResult(raw: BulkPortableExportResult): BulkPortableExportResult {
  return { ...raw, operation: normalizeLifecycleOperation(raw.operation), exports: arrayOrEmpty(raw.exports) }
}

function normalizeStorageManagementState(raw: StorageManagementState): StorageManagementState {
  const inv = raw.inventory
  return {
    ...raw,
    inventory: {
      ...inv,
      profiles: arrayOrEmpty(inv?.profiles),
      orphans: arrayOrEmpty(inv?.orphans),
      unsafe: arrayOrEmpty(inv?.unsafe),
      limitations: arrayOrEmpty(inv?.limitations),
    },
    limitations: arrayOrEmpty(raw.limitations),
  }
}

function normalizeStorageManagementReview(raw: StorageManagementReview): StorageManagementReview {
  return { state: normalizeStorageManagementState(raw.state), repairPlans: arrayOrEmpty(raw.repairPlans) }
}

export const multiProfileAPI = {
  isNative: () => Boolean(native()),
  updateMetadata: async (request: BulkMetadataUpdateRequest) => normalizeBulkMetadataUpdateResult(await requireNative().BulkUpdateProfileMetadata(request)),
  refreshHealth: async (request: BulkHealthRefreshRequest) => normalizeBulkHealthRefreshResult(await requireNative().BulkRefreshProfileHealth(request)),
  applyLifecycle: async (request: BulkLifecycleRequest) => normalizeBulkLifecycleResult(await requireNative().BulkApplyProfileLifecycle(request)),
  pickExportDirectory: () => requireNative().PickPortableExportDirectory(),
  exportProfiles: async (request: BulkPortableExportRequest) => normalizeBulkPortableExportResult(await requireNative().BulkExportPortableProfiles(request)),
  refreshStorage: async () => normalizeStorageManagementState(await requireNative().RefreshStorageManagement()),
  reviewStorage: async () => normalizeStorageManagementReview(await requireNative().ReviewStorageManagement()),
  pickOperationReportFile: (operationId: string) => requireNative().PickOperationReportFile(operationId),
  exportOperationReport: (request: OperationReportExportRequest) => requireNative().ExportLifecycleOperationReport(request),
}
