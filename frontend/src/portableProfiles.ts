import type { FingerprintConfig, Profile } from './types'

export type IdentityMode = 'new-identity' | 'preserve-identity'

export interface PortableKernelRequirement {
  provider: string
  version: string
  sha256: string
  sizeBytes: number
}

export interface PortableAdapterRequirement {
  kind: string
  version: string
  sha256: string
  sizeBytes: number
}

export interface PortablePayload {
  name: string
  group?: string
  notes?: string
  tags?: string[]
  fingerprint: FingerprintConfig
  proxyUrl?: string
  routeOmitted?: boolean
  credentialRequired?: boolean
  kernel: PortableKernelRequirement
  adapter?: PortableAdapterRequirement
  identityMode: IdentityMode
}

export interface PortableArtifact {
  schemaVersion: number
  kind: string
  applicationVersion: string
  exportedAt: string
  payload: PortablePayload
  payloadSha256: string
  exclusions: string[]
  limitations?: string[]
}

export interface PortableDependencyOption {
  id: string
  name: string
  kind: string
  version: string
  sha256: string
}

export interface PortableImportPreview {
  path: string
  artifact: PortableArtifact
  kernelMatches: PortableDependencyOption[]
  adapterMatches: PortableDependencyOption[]
  credentialRequired: boolean
  warnings: string[]
  ready: boolean
}

export interface PortableExportRequest {
  profileId: string
  destination: string
  identityMode: IdentityMode
}

export interface PortableExportResult {
  path: string
  profileId: string
  profileName: string
  identityMode: IdentityMode
  payloadSha256: string
  exportedAt: string
  exclusions: string[]
  limitations: string[]
}

export interface PortableImportRequest {
  path: string
  name?: string
  identityMode: IdentityMode
  kernelId: string
  adapterId?: string
  credentialId?: string
}

export interface PortableImportResult {
  profile: Profile
  identityMode: IdentityMode
  warnings: string[]
}

export interface PortableTemplate {
  schemaVersion: number
  id: string
  name: string
  payload: PortablePayload
  createdAt: string
  updatedAt: string
}

export interface PortableTemplateCreateRequest {
  profileId: string
  name: string
}

export interface PortableTemplateUpdateRequest {
  templateId: string
  name: string
  profileName: string
  group?: string
  notes?: string
  tags?: string[]
}

export interface PortableTemplateApplyRequest {
  templateId: string
  name?: string
  kernelId: string
  adapterId?: string
  credentialId?: string
}

type NativePortableAPI = {
  PickPortableExportFile(profileName: string): Promise<string>
  PickPortableImportFile(): Promise<string>
  ExportPortableProfile(request: PortableExportRequest): Promise<PortableExportResult>
  PreviewPortableImport(path: string): Promise<PortableImportPreview>
  ImportPortableProfile(request: PortableImportRequest): Promise<PortableImportResult>
  ListPortableTemplates(): Promise<PortableTemplate[]>
  CreatePortableTemplate(request: PortableTemplateCreateRequest): Promise<PortableTemplate>
  UpdatePortableTemplate(request: PortableTemplateUpdateRequest): Promise<PortableTemplate>
  DeletePortableTemplate(templateId: string): Promise<void>
  ApplyPortableTemplate(request: PortableTemplateApplyRequest): Promise<PortableImportResult>
}

function native(): NativePortableAPI | undefined {
  return (window as unknown as { go?: { main?: { DesktopApp?: NativePortableAPI } } }).go?.main?.DesktopApp
}

function requireNative(): NativePortableAPI {
  const api = native()
  if (!api) throw new Error('Portable Profile actions require the Wails desktop runtime.')
  return api
}

function arr<T>(value: T[] | null | undefined): T[] {
  return value || []
}

function normalizePortableExportResult(raw: PortableExportResult): PortableExportResult {
  return { ...raw, exclusions: arr(raw.exclusions), limitations: arr(raw.limitations) }
}

function normalizePortableImportPreview(raw: PortableImportPreview): PortableImportPreview {
  return { ...raw, kernelMatches: arr(raw.kernelMatches), adapterMatches: arr(raw.adapterMatches), warnings: arr(raw.warnings), artifact: { ...raw.artifact, exclusions: arr(raw.artifact?.exclusions), limitations: arr(raw.artifact?.limitations), payload: { ...raw.artifact?.payload, tags: arr(raw.artifact?.payload?.tags) } } }
}

function normalizePortableImportResult(raw: PortableImportResult): PortableImportResult {
  return { ...raw, warnings: arr(raw.warnings) }
}

export const portableProfileAPI = {
  isNative: () => Boolean(native()),
  pickExport: (profileName: string) => requireNative().PickPortableExportFile(profileName),
  pickImport: () => requireNative().PickPortableImportFile(),
  exportProfile: async (request: PortableExportRequest) => normalizePortableExportResult(await requireNative().ExportPortableProfile(request)),
  previewImport: async (path: string) => normalizePortableImportPreview(await requireNative().PreviewPortableImport(path)),
  importProfile: async (request: PortableImportRequest) => normalizePortableImportResult(await requireNative().ImportPortableProfile(request)),
  templates: async () => native() ? arr(await native()!.ListPortableTemplates()) : [],
  createTemplate: async (request: PortableTemplateCreateRequest) => await requireNative().CreatePortableTemplate(request),
  updateTemplate: (request: PortableTemplateUpdateRequest) => requireNative().UpdatePortableTemplate(request),
  deleteTemplate: (templateId: string) => requireNative().DeletePortableTemplate(templateId),
  applyTemplate: async (request: PortableTemplateApplyRequest) => normalizePortableImportResult(await requireNative().ApplyPortableTemplate(request)),
}
