import type { LifecycleOperation, LifecycleRecord, LifecycleState } from './lifecycle'
import type { AdapterRecord, CredentialRecord, KernelRecord, Profile, RuntimeSession } from './types'

export type SnapshotStatus = 'pending' | 'staged' | 'verified' | 'invalid' | 'recovery-required'
export type TrashStatus = 'pending' | 'stored' | 'restoring' | 'cleanup-pending' | 'deleted' | 'recovery-required'

export interface LocalSnapshotRecord {
  schemaVersion: number
  snapshotId: string
  sourceProfileId: string
  manifestRef: string
  status: SnapshotStatus
  createdAt: string
  updatedAt: string
  verifiedAt?: string
  manifestDigest: string
  treeDigest: string
  fileCount: number
  totalBytes: number
  limitations?: string[]
  revision: number
}

export interface LocalTrashRecord {
  schemaVersion: number
  trashId: string
  profileId: string
  operatingSystem: string
  architecture: string
  originalState: LifecycleState
  originalManagedDir: string
  originalArchivedAt?: string
  originalSourceId?: string
  originalRecoveryCodes?: string[]
  originalLimitationCodes?: string[]
  trashRef: string
  dataPresent: boolean
  profileDefinitionDigest: string
  treeDigest: string
  fileCount: number
  totalBytes: number
  status: TrashStatus
  trashedAt: string
  retentionDeadline: string
  deletedAt?: string
  updatedAt: string
  limitations?: string[]
  revision: number
}

export interface LocalRecoveryProgress {
  operationId: string
  operationType: string
  profileId: string
  status: string
  stage: string
  filesProcessed: number
  filesTotal: number
  bytesProcessed: number
  bytesTotal: number
  cancellationAvailable: boolean
  updatedAt: string
}

export interface TrashReconciliationFinding {
  trashId: string
  profileId: string
  status: string
  sourceState: string
  trashState: string
  profileState: string
  stagingState?: string
  reasonCode: string
}

export interface TrashReconciliationReport {
  generatedAt: string
  findings?: TrashReconciliationFinding[]
  limitations?: string[]
}

export interface LocalRecoveryState {
  snapshots: LocalSnapshotRecord[]
  trash: LocalTrashRecord[]
  progress: LocalRecoveryProgress[]
  trashReconciliation: TrashReconciliationReport
}

export interface LocalRecoveryPreflight {
  profileId: string
  lifecycleState: LifecycleState
  storageStatus: string
  active: boolean
  locked: boolean
  snapshotAllowed: boolean
  archiveAllowed: boolean
  unarchiveAllowed: boolean
  trashAllowed: boolean
  restoreTrashAllowed: boolean
  permanentDeleteAllowed: boolean
  trashId?: string
  retentionDeadline?: string
  reasons?: string[]
}

export interface RecoveryWorkspaceData {
  profiles: Profile[]
  lifecycleRecords: LifecycleRecord[]
  lifecycleOperations: LifecycleOperation[]
  sessions: RuntimeSession[]
  kernels: KernelRecord[]
  adapters: AdapterRecord[]
  credentials: CredentialRecord[]
}

export interface CreateLocalSnapshotRequest { profileId: string; idempotencyKey: string }
export interface RestoreLocalSnapshotRequest {
  snapshotId: string
  name?: string
  kernelId?: string
  adapterId?: string
  credentialId?: string
  idempotencyKey: string
}
export interface ArchiveProfileRequest { profileId: string; idempotencyKey: string }
export interface TrashProfileRequest { profileId: string; retentionDays: number; idempotencyKey: string }
export interface TrashProfileActionRequest {
  profileId: string
  trashId: string
  confirmation?: string
  idempotencyKey: string
}

type NativeRecoveryAPI = {
  LocalRecoveryState(): Promise<LocalRecoveryState>
  LocalRecoveryPreflight(profileId: string): Promise<LocalRecoveryPreflight>
  RefreshLocalRecovery(): Promise<LocalRecoveryState>
  CreateLocalSnapshot(request: CreateLocalSnapshotRequest): Promise<LocalRecoveryState>
  RestoreLocalSnapshot(request: RestoreLocalSnapshotRequest): Promise<LocalRecoveryState>
  ArchiveProfile(request: ArchiveProfileRequest): Promise<LocalRecoveryState>
  UnarchiveProfile(request: ArchiveProfileRequest): Promise<LocalRecoveryState>
  TrashProfile(request: TrashProfileRequest): Promise<LocalRecoveryState>
  RestoreTrashedProfile(request: TrashProfileActionRequest): Promise<LocalRecoveryState>
  PermanentlyDeleteTrashedProfile(request: TrashProfileActionRequest): Promise<LocalRecoveryState>
  CancelLocalRecoveryOperation(operationId: string): Promise<LocalRecoveryState>
}

const emptyState: LocalRecoveryState = {
  snapshots: [],
  trash: [],
  progress: [],
  trashReconciliation: { generatedAt: '', findings: [], limitations: [] },
}

function native(): NativeRecoveryAPI | undefined {
  return (window as unknown as { go?: { main?: { DesktopApp?: NativeRecoveryAPI } } }).go?.main?.DesktopApp
}

function requireNative(): NativeRecoveryAPI {
  const api = native()
  if (!api) throw new Error('Local recovery requires the Wails desktop runtime.')
  return api
}

export const localRecoveryAPI = {
  isNative: () => Boolean(native()),
  emptyState: () => ({ ...emptyState, snapshots: [], trash: [], progress: [], trashReconciliation: { ...emptyState.trashReconciliation, findings: [], limitations: [] } }),
  state: async () => native() ? native()!.LocalRecoveryState() : localRecoveryAPI.emptyState(),
  preflight: (profileId: string) => requireNative().LocalRecoveryPreflight(profileId),
  refresh: () => requireNative().RefreshLocalRecovery(),
  snapshot: (request: CreateLocalSnapshotRequest) => requireNative().CreateLocalSnapshot(request),
  restoreSnapshot: (request: RestoreLocalSnapshotRequest) => requireNative().RestoreLocalSnapshot(request),
  archive: (request: ArchiveProfileRequest) => requireNative().ArchiveProfile(request),
  unarchive: (request: ArchiveProfileRequest) => requireNative().UnarchiveProfile(request),
  trash: (request: TrashProfileRequest) => requireNative().TrashProfile(request),
  restoreTrash: (request: TrashProfileActionRequest) => requireNative().RestoreTrashedProfile(request),
  permanentDelete: (request: TrashProfileActionRequest) => requireNative().PermanentlyDeleteTrashedProfile(request),
  cancel: (operationId: string) => requireNative().CancelLocalRecoveryOperation(operationId),
}

export function newRecoveryKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  return `request-${Date.now()}-${Math.random().toString(16).slice(2)}`
}
