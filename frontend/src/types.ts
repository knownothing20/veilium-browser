export interface KernelRef {
  id?: string
  provider: string
  version: string
  executable: string
}

export interface KernelRecord {
  id: string
  name: string
  provider: string
  version: string
  executable: string
  sha256: string
  sizeBytes: number
  status: 'verified' | 'modified' | 'missing'
  importedAt: string
  verifiedAt: string
  packageRoot?: string
  packageTreeSha256?: string
  packageFileCount?: number
  packageSizeBytes?: number
  snapshotRevision?: number
  archiveSha256?: string
}

export interface KernelReleasePin {
  providerId: string
  providerRevision: number
  name: string
  browserVersion: string
  snapshotRevision: number
  sourceProject: string
  sourcePageUrl: string
  licenseSpdx: string
  licenseUrl: string
  thirdPartyNoticesUrl: string
  reviewedAt: string
  platform: string
  arch: string
  archiveName: string
  archiveUrl: string
  archiveSizeBytes: number
  archiveSha256: string
  archiveEntryCount: number
  archiveRoot: string
  executablePath: string
  executableSizeBytes: number
  executableSha256: string
  packageFileCount: number
  expandedSizeBytes: number
  packageTreeSha256: string
  limitations: string[]
}

export interface KernelInstallRequest {
  providerId: string
  version: string
  licenseAccepted: boolean
}

export interface KernelImportRequest {
  name: string
  provider: string
  version: string
  sourcePath: string
}

export interface AdapterRecord {
  id: string
  name: string
  kind: 'xray' | 'sing-box'
  version: string
  executable: string
  sha256: string
  sizeBytes: number
  licenseSpdx: string
  sourceUrl: string
  protocols: string[]
  status: 'verified' | 'modified' | 'missing'
  importedAt: string
  verifiedAt: string
  official: boolean
  officialTag?: string
  officialAsset?: string
  officialPlatform?: string
  officialArch?: string
}

export interface AdapterReleasePin {
  kind: 'xray' | 'sing-box'
  version: string
  tag: string
  repository: string
  licenseSpdx: string
  publishedAt: string
  platform: string
  arch: string
  assetName: string
  assetUrl: string
  archiveSha256: string
  archiveSizeBytes: number
  executablePath: string
  executableSha256: string
  executableSizeBytes: number
  versionArgs: string[]
  configurationArgs: string[]
}

export interface AdapterValidationCheck {
  id: string
  label: string
  status: 'pass' | 'warn' | 'fail'
  detail: string
}

export interface AdapterValidationReport {
  adapterId: string
  adapterName: string
  kind: string
  version: string
  officialTag: string
  platform: string
  arch: string
  status: 'passed' | 'failed'
  versionText: string
  checks: AdapterValidationCheck[]
  completedAt: string
}

export interface AdapterImportRequest {
  name: string
  kind: 'xray' | 'sing-box'
  version: string
  sourcePath: string
  licenseSpdx: string
  sourceUrl: string
}

export interface AdapterInstallRequest {
  kind: 'xray' | 'sing-box'
  version: string
  licenseAccepted: boolean
}

export interface CredentialRecord {
  id: string
  name: string
  username: string
  createdAt: string
  updatedAt: string
}

export interface CredentialSaveRequest {
  id?: string
  name: string
  username: string
  secret?: string
}

export type RuntimeState = 'starting' | 'ready' | 'stopping' | 'exited' | 'failed'

export interface RuntimeSession {
  profileId: string
  profileName: string
  state: RuntimeState
  pid: number
  cdpPort: number
  cdpUrl: string
  webSocketDebuggerUrl?: string
  browser?: string
  startedAt: string
  readyAt?: string
  exitedAt?: string
  exitCode?: number
  lastError?: string
  logPath: string
}

export type DiagnosticStatus = 'healthy' | 'degraded' | 'failed'
export type DiagnosticCheckStatus = 'pass' | 'warn' | 'fail' | 'skipped'

export interface ProxyDiagnosticCheck {
  id: string
  label: string
  status: DiagnosticCheckStatus
  detail: string
  latencyMs?: number
}

export interface ProxyDiagnosticReport {
  profileId: string
  profileName: string
  status: DiagnosticStatus
  proxyDisplay: string
  routeKind: string
  bridgeKind?: string
  exitIp?: string
  firstByteLatencyMs?: number
  totalLatencyMs?: number
  startedAt: string
  completedAt: string
  checks: ProxyDiagnosticCheck[]
  limitations: string[]
}

export interface FingerprintConfig {
  seed?: string
  platform: 'windows' | 'linux' | 'macos'
  brand: 'Chromium' | 'Chrome' | 'Edge' | 'Opera' | 'Vivaldi'
  language: string
  timezone: string
  screenWidth: number
  screenHeight: number
  hardwareConcurrency?: number
  deviceMemoryGb?: number
  webrtcPolicy: 'default' | 'proxy-only' | 'disabled'
  canvasMode: 'seeded' | 'native'
  audioMode: 'seeded' | 'native'
  fontMode: 'seeded' | 'native'
  clientRectsMode: 'seeded' | 'native'
  gpuProfile: 'auto' | 'native' | 'custom'
  gpuVendor?: string
  gpuRenderer?: string
}

export interface ProxyConfig {
  url?: string
  credentialRef?: string
  adapterRef?: string
}

export interface Profile {
  id: string
  name: string
  group?: string
  notes?: string
  kernel: KernelRef
  fingerprint: FingerprintConfig
  proxy: ProxyConfig
  userDataDir: string
  tags?: string[]
  createdAt?: string
  updatedAt?: string
}

export type ProviderTrustStatus = 'reviewed' | 'custom' | 'legacy' | 'disabled' | 'invalid'
export type CapabilityStatus = 'verified' | 'partial' | 'unsupported' | 'unverified' | 'failed'
export type CapabilityID =
  | 'platform'
  | 'browser-brand'
  | 'timezone'
  | 'surface-seed'
  | 'surface-controls'
  | 'hardware-concurrency'
  | 'device-memory'
  | 'custom-gpu'
  | 'webrtc-policy'

export interface CapabilityDeclaration {
  id: CapabilityID
  status: CapabilityStatus
  evidenceRequired: boolean
  minMajor?: number
  maxMajor?: number
  limitation?: string
}

export interface Capabilities {
  schemaVersion: number
  provider: string
  revision: number
  trustStatus: ProviderTrustStatus
  majorVersion: number
  capabilities: Partial<Record<CapabilityID, CapabilityDeclaration>>
  limitations?: string[]
}

export interface ProviderDescriptor {
  id: string
  name: string
  description: string
  versions: string[]
  samples: Capabilities[]
}

export interface Bootstrap {
  version: string
  profiles: Profile[]
  providers: ProviderDescriptor[]
  kernels: KernelRecord[]
  adapters: AdapterRecord[]
  sessions: RuntimeSession[]
  credentials: CredentialRecord[]
  credentialProvider: string
  adapterPins: AdapterReleasePin[]
  kernelPins: KernelReleasePin[]
  runtimePlatform: string
  runtimeArch: string
}

export type EvidenceRunStatus = 'pending' | 'running' | 'passed' | 'partial' | 'failed' | 'cancelled' | 'incomplete'
export type EvidenceObservationStatus = 'passed' | 'partial' | 'failed' | 'unavailable' | 'skipped'
export type EvidenceContext = 'top-level' | 'iframe' | 'worker'

export interface ProviderBinaryIdentity {
  schemaVersion: number
  providerId: string
  providerRevision: number
  providerTrust: ProviderTrustStatus
  browserVersion: string
  operatingSystem: string
  architecture: string
  executablePath: string
  executableSize: number
  executableSha256: string
  packageRoot?: string
  packageTreeSha256?: string
  packageFileCount?: number
  packageSizeBytes?: number
  snapshotRevision?: number
  archiveSha256?: string
  integrityStatus: string
  verificationTimestamp?: string
  provenance: string
  reviewed: boolean
  limitations?: string[]
}

export interface EvidenceObservation {
  id: string
  context: EvidenceContext
  capabilityId?: CapabilityID
  status: EvidenceObservationStatus
  expected?: string
  observed?: string
  reasonCode?: string
  detail?: string
}

export interface EvidenceRun {
  schemaVersion: number
  id: string
  profileId: string
  profileName: string
  providerId: string
  providerRevision: number
  providerTrust: ProviderTrustStatus
  binaryIdentity: ProviderBinaryIdentity
  browserVersion: string
  operatingSystem: string
  architecture: string
  harnessRevision: string
  status: EvidenceRunStatus
  startedAt: string
  completedAt?: string
  expiresAt: string
  observations: EvidenceObservation[]
  limitations?: string[]
  failureCode?: string
  failureDetail?: string
}

export interface LaunchPlan {
  executable: string
  args: string[]
  environment?: Record<string, string>
  proxyDisplay: string
  requiresBridge: boolean
  bridgeKind?: string
  warnings?: string[]
}
