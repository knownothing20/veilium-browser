import { defaultProfile } from './model'
import type {
  Bootstrap,
  Capabilities,
  KernelImportRequest,
  KernelRecord,
  LaunchPlan,
  Profile,
  ProviderDescriptor,
  RuntimeSession,
} from '../types'

type WailsDesktopApp = {
  Bootstrap: () => Promise<Bootstrap>
  ListProfiles: () => Promise<Profile[]>
  ListSessions: () => Promise<RuntimeSession[]>
  Capabilities: (provider: string, version: string) => Promise<Capabilities>
  CreateProfile: (profile: Profile) => Promise<Profile>
  UpdateProfile: (profile: Profile) => Promise<Profile>
  CloneProfile: (id: string, name: string) => Promise<Profile>
  DeleteProfile: (id: string) => Promise<void>
  BuildLaunchPlan: (request: { profileId: string; remoteDebuggingPort: number }) => Promise<LaunchPlan>
  StartProfile: (profileId: string) => Promise<RuntimeSession>
  StopProfile: (profileId: string) => Promise<RuntimeSession>
  PickKernelExecutable: () => Promise<string>
  ImportKernel: (request: KernelImportRequest) => Promise<KernelRecord>
  VerifyKernel: (id: string) => Promise<KernelRecord>
  DeleteKernel: (id: string) => Promise<void>
}

declare global {
  interface Window {
    go?: { main?: { DesktopApp?: WailsDesktopApp } }
  }
}

const providers: ProviderDescriptor[] = [
  {
    id: 'patched-chromium',
    name: 'Patched Chromium',
    description: 'Verified, version-aware fingerprint controls.',
    versions: ['148.0.0', '144.0.0', '142.0.0'],
    samples: [{ provider: 'patched-chromium', majorVersion: 148, canSetPlatform: true, canSetBrand: true, canSetTimezone: true, canSeedSurfaces: true, canDisableSurfaces: true, canSetHardwareThreads: true, canSetDeviceMemory: false, canSetCustomGpu: false, supportsProxyOnlyWebRtc: true }],
  },
  {
    id: 'native-chromium',
    name: 'Native Chromium',
    description: 'Profile isolation without synthetic surface modification.',
    versions: ['148.0.0', '144.0.0'],
    samples: [{ provider: 'native-chromium', majorVersion: 148, canSetPlatform: false, canSetBrand: false, canSetTimezone: false, canSeedSurfaces: false, canDisableSurfaces: false, canSetHardwareThreads: false, canSetDeviceMemory: false, canSetCustomGpu: false, supportsProxyOnlyWebRtc: true }],
  },
]

let mockKernels: KernelRecord[] = []
let mockProfiles: Profile[] = []
let mockSessions: RuntimeSession[] = []

function native(): WailsDesktopApp | undefined {
  return window.go?.main?.DesktopApp
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

export const backend = {
  isNative: () => Boolean(native()),

  async bootstrap(): Promise<Bootstrap> {
    const api = native()
    if (api) return api.Bootstrap()
    return {
      version: '0.4.0-browser-preview',
      profiles: clone(mockProfiles),
      providers: clone(providers),
      kernels: clone(mockKernels),
      sessions: clone(mockSessions),
    }
  },

  async listSessions(): Promise<RuntimeSession[]> {
    const api = native()
    if (api) return api.ListSessions()
    return clone(mockSessions)
  },

  async capabilities(provider: string, version: string): Promise<Capabilities> {
    const api = native()
    if (api) return api.Capabilities(provider, version)
    const descriptor = providers.find((item) => item.id === provider)
    const major = Number.parseInt(version, 10)
    const sample = descriptor?.samples.find((item) => item.majorVersion === major) ?? descriptor?.samples[0]
    if (!sample) throw new Error(`Unknown provider ${provider}`)
    return clone(sample)
  },

  async createProfile(input: Profile): Promise<Profile> {
    const api = native()
    if (api) return api.CreateProfile(input)
    const created = { ...clone(input), id: crypto.randomUUID(), createdAt: new Date().toISOString(), updatedAt: new Date().toISOString() }
    mockProfiles = [...mockProfiles, created]
    return clone(created)
  },

  async updateProfile(input: Profile): Promise<Profile> {
    const api = native()
    if (api) return api.UpdateProfile(input)
    const updated = { ...clone(input), updatedAt: new Date().toISOString() }
    mockProfiles = mockProfiles.map((item) => item.id === input.id ? updated : item)
    return clone(updated)
  },

  async cloneProfile(id: string, name: string): Promise<Profile> {
    const api = native()
    if (api) return api.CloneProfile(id, name)
    const source = mockProfiles.find((item) => item.id === id)
    if (!source) throw new Error('Profile not found')
    const cloned = { ...clone(source), id: crypto.randomUUID(), name: name || `${source.name} Copy`, createdAt: new Date().toISOString() }
    mockProfiles = [...mockProfiles, cloned]
    return clone(cloned)
  },

  async deleteProfile(id: string): Promise<void> {
    const api = native()
    if (api) return api.DeleteProfile(id)
    mockProfiles = mockProfiles.filter((item) => item.id !== id)
  },

  async buildLaunchPlan(profileId: string, remoteDebuggingPort = 0): Promise<LaunchPlan> {
    const api = native()
    if (api) return api.BuildLaunchPlan({ profileId, remoteDebuggingPort })
    const profile = mockProfiles.find((item) => item.id === profileId)
    if (!profile) throw new Error('Profile not found')
    return {
      executable: profile.kernel.executable,
      proxyDisplay: profile.proxy.url || 'direct://',
      requiresBridge: Boolean(profile.proxy.credentialRef),
      bridgeKind: profile.proxy.credentialRef ? 'local-auth-bridge' : undefined,
      args: [`--user-data-dir=${profile.userDataDir}`, '--remote-debugging-address=127.0.0.1'],
      warnings: ['Browser preview mode: no native process will be launched.'],
    }
  },

  async startProfile(profileId: string): Promise<RuntimeSession> {
    const api = native()
    if (!api) throw new Error('Browser process execution is available only in the Wails desktop application')
    return api.StartProfile(profileId)
  },

  async stopProfile(profileId: string): Promise<RuntimeSession> {
    const api = native()
    if (!api) throw new Error('Browser process execution is available only in the Wails desktop application')
    return api.StopProfile(profileId)
  },

  async pickKernelExecutable(): Promise<string> {
    const api = native()
    if (!api) throw new Error('File selection is available in the Wails desktop application')
    return api.PickKernelExecutable()
  },

  async importKernel(request: KernelImportRequest): Promise<KernelRecord> {
    const api = native()
    if (!api) throw new Error('Kernel import is available in the Wails desktop application')
    return api.ImportKernel(request)
  },

  async verifyKernel(id: string): Promise<KernelRecord> {
    const api = native()
    if (api) return api.VerifyKernel(id)
    const record = mockKernels.find((item) => item.id === id)
    if (!record) throw new Error('Kernel not found')
    return clone(record)
  },

  async deleteKernel(id: string): Promise<void> {
    const api = native()
    if (api) return api.DeleteKernel(id)
    if (mockProfiles.some((item) => item.kernel.id === id)) throw new Error('Kernel is used by a profile')
    mockKernels = mockKernels.filter((item) => item.id !== id)
  },
}

export { providers, defaultProfile }
