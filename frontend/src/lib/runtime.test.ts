import { describe, expect, it } from 'vitest'
import { isRuntimeActive, runtimeStateLabel, sessionForProfile } from './runtime'
import type { RuntimeSession } from '../types'

const session: RuntimeSession = {
  profileId: 'profile-a',
  profileName: 'Profile A',
  state: 'ready',
  pid: 123,
  cdpPort: 9222,
  cdpUrl: 'http://127.0.0.1:9222',
  startedAt: new Date().toISOString(),
  logPath: '/logs/profile-a.log',
}

describe('runtime model', () => {
  it('finds sessions and classifies active states', () => {
    expect(sessionForProfile([session], 'profile-a')).toEqual(session)
    expect(isRuntimeActive(session)).toBe(true)
    expect(isRuntimeActive({ ...session, state: 'failed' })).toBe(true)
    expect(isRuntimeActive({ ...session, state: 'failed', exitedAt: new Date().toISOString() })).toBe(false)
  })

  it('labels missing sessions as stopped', () => {
    expect(runtimeStateLabel()).toBe('stopped')
  })
})
