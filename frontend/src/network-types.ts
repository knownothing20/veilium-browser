import type { ProviderTrustStatus } from './types'

export type NetworkRunStatus = 'pending' | 'running' | 'passed' | 'partial' | 'failed' | 'unavailable' | 'cancelled' | 'incomplete'
export type NetworkObservationStatus = 'passed' | 'partial' | 'failed' | 'unavailable' | 'skipped'
export type NetworkProbeKind = 'exit-ip' | 'webrtc-stun' | 'delegated-dns'
export type NetworkCompatibilityStatus = 'verified' | 'partial' | 'unsupported' | 'failed' | 'stale' | 'untested'

export interface NetworkProbeDefinition {
  schemaVersion: number
  id: string
  revision: number
  kind: NetworkProbeKind
  httpsUrl?: string
  stunServer?: string
  dnsZone?: string
  dnsResultUrl?: string
  timeoutSeconds: number
  maxResponseBytes?: number
  selfHostable: boolean
  privacyNote: string
}

export interface NetworkProbeSet {
  schemaVersion: number
  id: string
  revision: number
  definitions: NetworkProbeDefinition[]
}

export interface NetworkProbeConfiguration {
  configured: boolean
  probeSet: NetworkProbeSet
}

export interface NetworkObservation {
  id: string
  probeKind: NetworkProbeKind
  probeId: string
  probeRevision: number
  status: NetworkObservationStatus
  expected?: string
  values?: string[]
  reasonCode?: string
  detail?: string
  collectedAt: string
}

export interface NetworkEvidenceRun {
  schemaVersion: number
  id: string
  evidenceRunId: string
  profileId: string
  providerId: string
  providerRevision: number
  browserVersion: string
  operatingSystem: string
  architecture: string
  binaryIdentityDigest: string
  consistencyInputDigest: string
  route: { kind: string; scheme: string; bridgeKind?: string; digest: string }
  probeSetId: string
  probeSetRevision: number
  status: NetworkRunStatus
  startedAt: string
  completedAt?: string
  expiresAt: string
  observations: NetworkObservation[]
  limitations?: string[]
  failureCode?: string
  failureDetail?: string
}

export interface NetworkCompatibilityEntry {
  schemaVersion: number
  providerId: string
  providerRevision: number
  providerTrust: ProviderTrustStatus
  browserVersion: string
  operatingSystem: string
  architecture: string
  binaryIdentityDigest: string
  capabilityId: string
  status: NetworkCompatibilityStatus
  probeSetId?: string
  probeSetRevision?: number
  networkEvidenceIds?: string[]
  reviewedAt?: string
  evidenceExpiresAt?: string
  limitations?: string[]
}

export interface NetworkCompatibilityMatrix {
  schemaVersion: number
  generatedAt: string
  entries: NetworkCompatibilityEntry[]
}
