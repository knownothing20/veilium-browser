import type {
  Capabilities,
  CapabilityDeclaration,
  CapabilityID,
  CapabilityStatus,
  KernelRecord,
  Profile,
  ProviderDescriptor,
  ProviderTrustStatus,
} from '../types'

export type ProfileHealth = 'ready' | 'warning' | 'incomplete'

export function defaultProfile(provider?: ProviderDescriptor): Profile {
  const selectedProvider = provider?.id ?? 'custom-chromium'
  const version = provider?.versions[0] ?? '148.0.0'
  return {
    id: '',
    name: '',
    group: 'Default',
    notes: '',
    kernel: { provider: selectedProvider, version, executable: '' },
    fingerprint: {
      platform: 'windows',
      brand: 'Chromium',
      language: 'en-US',
      timezone: 'America/Los_Angeles',
      screenWidth: 1920,
      screenHeight: 1080,
      webrtcPolicy: 'proxy-only',
      canvasMode: 'native',
      audioMode: 'native',
      fontMode: 'native',
      clientRectsMode: 'native',
      gpuProfile: 'auto',
    },
    proxy: { url: 'direct://' },
    userDataDir: '',
    tags: [],
  }
}

export function applyKernel(profile: Profile, record: KernelRecord): Profile {
  return {
    ...profile,
    kernel: { id: record.id, provider: record.provider, version: record.version, executable: record.executable },
  }
}

export function filterProfiles(profiles: Profile[], query: string, group: string): Profile[] {
  const normalized = query.trim().toLowerCase()
  return profiles.filter((profile) => {
    if (group && group !== 'all' && (profile.group || 'Default') !== group) return false
    if (!normalized) return true
    return [profile.name, profile.group, profile.kernel.provider, profile.kernel.version, profile.proxy.url, ...(profile.tags || [])]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
      .includes(normalized)
  })
}

export function profileHealth(profile: Profile): ProfileHealth {
  if (!profile.name.trim() || !profile.kernel.executable.trim()) return 'incomplete'
  if (profile.fingerprint.webrtcPolicy === 'default' && profile.proxy.url !== 'direct://') return 'warning'
  if (profile.proxy.url && profile.proxy.url !== 'direct://' && profile.proxy.url.includes('@')) return 'warning'
  return 'ready'
}

export function capabilitiesFor(
  providers: ProviderDescriptor[],
  providerID: string,
  version: string,
): Capabilities | undefined {
  const provider = providers.find((item) => item.id === providerID)
  const major = Number.parseInt(version.split('.')[0] || '0', 10)
  return provider?.samples.find((sample) => sample.majorVersion === major) ?? provider?.samples[0]
}

export function capabilityFor(
  providers: ProviderDescriptor[],
  providerID: string,
  version: string,
  capabilityID: CapabilityID,
): CapabilityDeclaration | undefined {
  const capabilities = capabilitiesFor(providers, providerID, version)
  const declaration = capabilities?.capabilities[capabilityID]
  if (!declaration) return undefined
  if (declaration.minMajor && capabilities && capabilities.majorVersion < declaration.minMajor) return undefined
  if (declaration.maxMajor && capabilities && capabilities.majorVersion > declaration.maxMajor) return undefined
  return declaration
}

export function capabilityStatus(
  providers: ProviderDescriptor[],
  providerID: string,
  version: string,
  capabilityID: CapabilityID,
): CapabilityStatus {
  return capabilityFor(providers, providerID, version, capabilityID)?.status ?? 'unsupported'
}

export function capabilityAllowsConfiguration(status: CapabilityStatus): boolean {
  return status === 'verified' || status === 'partial'
}

export function providerTrust(
  providers: ProviderDescriptor[],
  providerID: string,
  version: string,
): ProviderTrustStatus {
  return capabilitiesFor(providers, providerID, version)?.trustStatus ?? 'invalid'
}

export function capabilityLabel(status: CapabilityStatus): string {
  switch (status) {
    case 'verified': return 'Verified'
    case 'partial': return 'Partial'
    case 'unverified': return 'Unverified'
    case 'failed': return 'Failed'
    default: return 'Unsupported'
  }
}

export function groupsOf(profiles: Profile[]): string[] {
  return Array.from(new Set(profiles.map((profile) => profile.group || 'Default'))).sort((a, b) => a.localeCompare(b))
}
