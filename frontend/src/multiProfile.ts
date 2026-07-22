import type { LifecycleOperation, LifecycleOperationStatus, LifecycleState, StorageInventory } from './lifecycle'
import type { IdentityMode, PortableExportResult } from './portableProfiles'
import type { Profile } from './types'

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

export const multiProfileAPI = {
  isNative: () => Boolean(native()),
  updateMetadata: (request: BulkMetadataUpdateRequest) => requireNative().BulkUpdateProfileMetadata(request),
  refreshHealth: (request: BulkHealthRefreshRequest) => requireNative().BulkRefreshProfileHealth(request),
  applyLifecycle: (request: BulkLifecycleRequest) => requireNative().BulkApplyProfileLifecycle(request),
  pickExportDirectory: () => requireNative().PickPortableExportDirectory(),
  exportProfiles: (request: BulkPortableExportRequest) => requireNative().BulkExportPortableProfiles(request),
  refreshStorage: () => requireNative().RefreshStorageManagement(),
  reviewStorage: () => requireNative().ReviewStorageManagement(),
  pickOperationReportFile: (operationId: string) => requireNative().PickOperationReportFile(operationId),
  exportOperationReport: (request: OperationReportExportRequest) => requireNative().ExportLifecycleOperationReport(request),
}
