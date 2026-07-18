import { describe, expect, it } from 'vitest'
import {
  applyKernel,
  capabilityAllowsConfiguration,
  capabilityStatus,
  defaultProfile,
  filterProfiles,
  profileHealth,
  providerTrust,
} from './model'
import type { ProviderDescriptor } from '../types'

const providers: ProviderDescriptor[] = [
  {
    id: 'custom-chromium',
    name: 'Custom local Chromium',
    description: 'Generic custom provider',
    versions: ['148.0.0'],
    samples: [{
      schemaVersion: 2,
      provider: 'custom-chromium',
      revision: 1,
      trustStatus: 'custom',
      majorVersion: 148,
      capabilities: {
        'surface-seed': {
          id: 'surface-seed',
          status: 'unsupported',
          evidenceRequired: false,
        },
        'webrtc-policy': {
          id: 'webrtc-policy',
          status: 'unverified',
          evidenceRequired: true,
        },
      },
    }],
  },
]

const profiles = [
  { ...defaultProfile(providers[0]), id: 'a', name: 'Shop US', group: 'Commerce', tags: ['amazon'] },
  { ...defaultProfile(providers[0]), id: 'b', name: 'Social JP', group: 'Social', tags: ['x'] },
]

describe('profile model', () => {
  it('filters by query and group', () => {
    expect(filterProfiles(profiles, 'amazon', 'all')).toHaveLength(1)
    expect(filterProfiles(profiles, '', 'Social')).toEqual([profiles[1]])
  })

  it('marks missing executable as incomplete', () => {
    expect(profileHealth(defaultProfile(providers[0]))).toBe('incomplete')
  })

  it('creates generic defaults without advanced fingerprint claims', () => {
    const profile = defaultProfile(providers[0])
    expect(profile.kernel.provider).toBe('custom-chromium')
    expect(profile.fingerprint.brand).toBe('Chromium')
    expect(profile.fingerprint.canvasMode).toBe('native')
    expect(profile.fingerprint.hardwareConcurrency).toBeUndefined()
    expect(profile.fingerprint.webrtcPolicy).toBe('proxy-only')
  })

  it('applies an integrity-verified registry record atomically', () => {
    const profile = applyKernel(defaultProfile(providers[0]), {
      id: 'k1',
      name: 'Chromium',
      provider: 'custom-chromium',
      version: '148.0.0',
      executable: '/managed/chrome',
      sha256: 'a'.repeat(64),
      sizeBytes: 1,
      status: 'verified',
      importedAt: '',
      verifiedAt: '',
    })
    expect(profile.kernel).toEqual({ id: 'k1', provider: 'custom-chromium', version: '148.0.0', executable: '/managed/chrome' })
  })

  it('keeps trust and capability state separate', () => {
    expect(providerTrust(providers, 'custom-chromium', '148.0.0')).toBe('custom')
    expect(capabilityStatus(providers, 'custom-chromium', '148.0.0', 'surface-seed')).toBe('unsupported')
    expect(capabilityStatus(providers, 'custom-chromium', '148.0.0', 'webrtc-policy')).toBe('unverified')
    expect(capabilityAllowsConfiguration('unverified')).toBe(false)
    expect(capabilityAllowsConfiguration('partial')).toBe(true)
  })
})
