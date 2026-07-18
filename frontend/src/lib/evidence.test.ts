import { describe, expect, it } from 'vitest'
import { evidenceStatusClass, evidenceStatusLabel, evidenceSummary, latestEvidence } from './evidence'
import type { EvidenceRun } from '../types'

function run(id: string, startedAt: string, status: EvidenceRun['status']): EvidenceRun {
  return {
    schemaVersion: 1,
    id,
    profileId: 'p1',
    profileName: 'Profile',
    providerId: 'custom-chromium',
    providerRevision: 1,
    providerTrust: 'custom',
    binaryIdentity: {
      schemaVersion: 1,
      providerId: 'custom-chromium',
      providerRevision: 1,
      providerTrust: 'custom',
      browserVersion: '148.0.0',
      operatingSystem: 'linux',
      architecture: 'amd64',
      executablePath: '/managed/chrome',
      executableSize: 1,
      executableSha256: 'a'.repeat(64),
      integrityStatus: 'verified',
      provenance: 'managed-local-import',
      reviewed: false,
    },
    browserVersion: '148.0.0',
    operatingSystem: 'linux',
    architecture: 'amd64',
    harnessRevision: 'm4.2-v1',
    status,
    startedAt,
    completedAt: startedAt,
    expiresAt: startedAt,
    observations: [
      { id: 'a', context: 'top-level', status: 'passed' },
      { id: 'b', context: 'iframe', status: 'partial' },
    ],
  }
}

describe('evidence helpers', () => {
  it('selects the newest run', () => {
    expect(latestEvidence([run('old', '2026-07-18T10:00:00Z', 'partial'), run('new', '2026-07-18T11:00:00Z', 'passed')])?.id).toBe('new')
  })

  it('maps terminal status for display', () => {
    expect(evidenceStatusLabel('incomplete')).toBe('Incomplete')
    expect(evidenceStatusClass('failed')).toBe('failed')
    expect(evidenceStatusClass('partial')).toBe('degraded')
  })

  it('summarizes observation counts', () => {
    expect(evidenceSummary(run('one', '2026-07-18T10:00:00Z', 'partial'))).toBe('1 passed · 1 partial')
  })
})
