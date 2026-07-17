import type { Capabilities, Profile, ProviderDescriptor } from '../types'

export type ProfileHealth = 'ready' | 'warning' | 'incomplete'

export function defaultProfile(provider?: ProviderDescriptor): Profile {
  const selectedProvider = provider?.id ?? 'patched-chromium'
  const version = provider?.versions[0] ?? '148.0.0'
  return {
    id: '',
    name: '',
    group: 'Default',
    notes: '',
    kernel: {
      provider: selectedProvider,
      version,
      executable: '',
    },
    fingerprint: {
      platform: 'windows',
      brand: selectedProvider === 'native-chromium' ? 'Chromium' : 'Chrome',
      language: 'en-US',
      timezone: 'America/Los_Angeles',
      screenWidth: 1920,
      screenHeight: 1080,
      hardwareConcurrency: 8,
      webrtcPolicy: 'proxy-only',
      canvasMode: selectedProvider === 'native-chromium' ? 'native' : 'seeded',
      audioMode: selectedProvider === 'native-chromium' ? 'native' : 'seeded',
      fontMode: selectedProvider === 'native-chromium' ? 'native' : 'seeded',
      clientRectsMode: selectedProvider === 'native-chromium' ? 'native' : 'seeded',
      gpuProfile: 'auto',
    },
    proxy: { url: 'direct://' },
    userDataDir: '',
    tags: [],
  }
}

export function filterProfiles(profiles: Profile[], query: string, group: string): Profile[] {
  const normalized = query.trim().toLowerCase()
  return profiles.filter((profile) => {
    if (group && group !== 'all' && (profile.group || 'Default') !== group) return false
    if (!normalized) return true
    const haystack = [
      profile.name,
      profile.group,
      profile.kernel.provider,
      profile.kernel.version,
      profile.proxy.url,
      ...(profile.tags || []),
    ]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
    return haystack.includes(normalized)
  })
}

export function profileHealth(profile: Profile): ProfileHealth {
  if (!profile.name.trim() || !profile.kernel.executable.trim()) return 'incomplete'
  if (profile.fingerprint.webrtcPolicy === 'default' && profile.proxy.url !== 'direct://') return 'warning'
  if (profile.proxy.url && profile.proxy.url !== 'direct://' && profile.proxy.url.includes('@')) return 'warning'
  return 'ready'
}

export function capabilityFor(
  providers: ProviderDescriptor[],
  providerID: string,
  version: string,
): Capabilities | undefined {
  const provider = providers.find((item) => item.id === providerID)
  const major = Number.parseInt(version.split('.')[0] || '0', 10)
  return provider?.samples.find((sample) => sample.majorVersion === major) ?? provider?.samples[0]
}

export function groupsOf(profiles: Profile[]): string[] {
  return Array.from(new Set(profiles.map((profile) => profile.group || 'Default'))).sort((a, b) =>
    a.localeCompare(b),
  )
}
