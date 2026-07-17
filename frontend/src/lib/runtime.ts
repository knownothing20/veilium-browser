import type { RuntimeSession, RuntimeState } from '../types'

const activeStates: RuntimeState[] = ['starting', 'ready', 'stopping']

export function isRuntimeActive(session?: RuntimeSession): boolean {
  return Boolean(session && (activeStates.includes(session.state) || (session.state === 'failed' && !session.exitedAt)))
}

export function sessionForProfile(sessions: RuntimeSession[], profileId: string): RuntimeSession | undefined {
  return sessions.find((session) => session.profileId === profileId)
}

export function runtimeStateLabel(state?: RuntimeState): string {
  if (!state) return 'stopped'
  return state
}
