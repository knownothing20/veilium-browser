import { describe, expect, it } from 'vitest'
import {
  lifecycleAllowsEdit,
  lifecycleAllowsLaunch,
  lifecycleLabel,
  normalizeLifecycleBootstrap,
  type LifecycleRecord,
} from './lifecycle'
import type { Bootstrap } from './types'

const bootstrap: Bootstrap = {
  version: 'test',
  profiles: [],
  providers: [],
  kernels: [],
  adapters: [],
  sessions: [],
  credentials: [],
  credentialProvider: 'test',
  adapterPins: [],
  kernelPins: [],
  runtimePlatform: 'linux',
  runtimeArch: 'amd64',
}

const available: LifecycleRecord = {
  schemaVersion: 1,
  profileId: 'profile-a',
  state: 'available',
  managedDir: 'profiles/profile-a',
  createdAt: '2026-07-20T00:00:00Z',
  updatedAt: '2026-07-20T00:00:00Z',
  revision: 1,
}

describe('lifecycle presentation policy', () => {
  it('normalizes browser preview bootstrap without lifecycle claims', () => {
    const normalized = normalizeLifecycleBootstrap(bootstrap)
    expect(normalized.lifecycleRecords).toEqual([])
    expect(normalized.lifecycleOperations).toEqual([])
    expect(normalized.lifecycleReconciliation.inventory.managedRoot).toBe('.')
  })

  it('allows launch only for available unlocked Profiles', () => {
    expect(lifecycleAllowsLaunch(available)).toBe(true)
    expect(lifecycleAllowsLaunch({ ...available, state: 'draft' })).toBe(false)
    expect(lifecycleAllowsLaunch({ ...available, lock: { operationId: 'op-1', acquiredAt: available.updatedAt } })).toBe(false)
    expect(lifecycleAllowsLaunch(undefined)).toBe(false)
  })

  it('blocks archived and trashed editing while keeping invalid state inspectable', () => {
    expect(lifecycleAllowsEdit(available)).toBe(true)
    expect(lifecycleAllowsEdit({ ...available, state: 'invalid' })).toBe(true)
    expect(lifecycleAllowsEdit({ ...available, state: 'archived' })).toBe(false)
    expect(lifecycleAllowsEdit({ ...available, state: 'trashed' })).toBe(false)
  })

  it('labels missing and locked lifecycle state explicitly', () => {
    expect(lifecycleLabel(undefined)).toBe('missing lifecycle')
    expect(lifecycleLabel({ ...available, lock: { operationId: 'op-1', acquiredAt: available.updatedAt } })).toBe('available · locked')
  })
})
