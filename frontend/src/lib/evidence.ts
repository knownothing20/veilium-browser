import type { EvidenceObservation, EvidenceRun, EvidenceRunStatus } from '../types'

const labels: Record<EvidenceRunStatus, string> = {
  pending: 'Pending',
  running: 'Running',
  passed: 'Passed',
  partial: 'Partial',
  failed: 'Failed',
  cancelled: 'Cancelled',
  incomplete: 'Incomplete',
}

export function evidenceStatusLabel(status: EvidenceRunStatus): string {
  return labels[status]
}

export function evidenceStatusClass(status: EvidenceRunStatus): string {
  if (status === 'passed') return 'healthy'
  if (status === 'failed') return 'failed'
  if (status === 'pending' || status === 'running') return 'running'
  return 'degraded'
}

export function latestEvidence(runs: EvidenceRun[]): EvidenceRun | undefined {
  return [...runs].sort((a, b) => Date.parse(b.startedAt) - Date.parse(a.startedAt))[0]
}

export function observationCounts(observations: EvidenceObservation[]): Record<string, number> {
  return observations.reduce<Record<string, number>>((counts, observation) => {
    counts[observation.status] = (counts[observation.status] || 0) + 1
    return counts
  }, {})
}

export function evidenceSummary(run: EvidenceRun): string {
  const counts = observationCounts(run.observations)
  const parts = [
    counts.passed ? `${counts.passed} passed` : '',
    counts.partial ? `${counts.partial} partial` : '',
    counts.failed ? `${counts.failed} failed` : '',
    counts.unavailable ? `${counts.unavailable} unavailable` : '',
  ].filter(Boolean)
  return parts.join(' · ') || 'No observations'
}
