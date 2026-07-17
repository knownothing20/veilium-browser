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
}

export interface AdapterImportRequest {
  name: string
  kind: 'xray' | 'sing-box'
  version: string
  sourcePath: string
  licenseSpdx: string
  sourceUrl: string
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

export interface Capabilities {
  provider: string
  majorVersion: number
  canSetPlatform: boolean
  canSetBrand: boolean
  canSetTimezone: boolean
  canSeedSurfaces: boolean
  canDisableSurfaces: boolean
  canSetHardwareThreads: boolean
  canSetDeviceMemory: boolean
  canSetCustomGpu: boolean
  supportsProxyOnlyWebRtc: boolean
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
