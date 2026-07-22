import type { LifecycleOperation, LifecycleState, StorageInventory } from './lifecycle'
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

type NativeMultiProfileAPI = {
  BulkUpdateProfileMetadata(request: BulkMetadataUpdateRequest): Promise<BulkMetadataUpdateResult>
  BulkRefreshProfileHealth(request: BulkHealthRefreshRequest): Promise<BulkHealthRefreshResult>
  RefreshStorageManagement(): Promise<StorageManagementState>
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
  refreshStorage: () => requireNative().RefreshStorageManagement(),
}
