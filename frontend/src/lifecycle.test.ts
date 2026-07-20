import { describe, expect, it } from 'vitest'
import {
  cancellationAvailability,
  lifecycleAllowsEdit,
  lifecycleAllowsLaunch,
  lifecycleLabel,
  normalizeLifecycleBootstrap,
  type LifecycleOperation,
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

const operation: LifecycleOperation = {
  schemaVersion: 1,
  id: 'operation-a',
  type: 'storage-reconcile',
  profileIds: ['profile-a'],
  status: 'running',
  stage: 'locked',
  startedAt: '2026-07-20T00:00:00Z',
  updatedAt: '2026-07-20T00:00:01Z',
  cancellationRequested: false,
  safeCancellationStage: 'between-items',
  applicationVersion: 'test',
  platform: 'linux',
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

  it('reports cancellation availability without exposing a mutation action', () => {
    expect(cancellationAvailability(operation)).toBe('Cancellation available at between-items')
    expect(cancellationAvailability({ ...operation, cancellationRequested: true })).toBe('Cancellation requested')
    expect(cancellationAvailability({ ...operation, status: 'completed' })).toBe('Cancellation unavailable · terminal')
    expect(cancellationAvailability({ ...operation, safeCancellationStage: '' })).toBe('Cancellation unavailable at current stage')
  })
})
