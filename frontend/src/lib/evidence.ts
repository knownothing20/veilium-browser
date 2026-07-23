import type { EvidenceObservation, EvidenceRun, EvidenceRunStatus } from '../types'

const labels: Record<EvidenceRunStatus, string> = {
  pending: '等待中',
  running: '进行中',
  passed: '通过',
  partial: '部分通过',
  failed: '失败',
  cancelled: '已取消',
  incomplete: '不完整',
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
    counts.passed ? `${counts.passed} 项通过` : '',
    counts.partial ? `${counts.partial} 项部分通过` : '',
    counts.failed ? `${counts.failed} 项失败` : '',
    counts.unavailable ? `${counts.unavailable} 项不可用` : '',
  ].filter(Boolean)
  return parts.join(' · ') || '没有观测结果'
}
