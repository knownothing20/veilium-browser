import { describe, expect, it } from 'vitest'
import { defaultProfile, filterProfiles, profileHealth } from './model'

const profiles = [
  { ...defaultProfile(), id: 'a', name: 'Shop US', group: 'Commerce', tags: ['amazon'] },
  { ...defaultProfile(), id: 'b', name: 'Social JP', group: 'Social', tags: ['x'] },
]

describe('profile model', () => {
  it('filters by query and group', () => {
    expect(filterProfiles(profiles, 'amazon', 'all')).toHaveLength(1)
    expect(filterProfiles(profiles, '', 'Social')).toEqual([profiles[1]])
  })

  it('marks missing executable as incomplete', () => {
    expect(profileHealth(defaultProfile())).toBe('incomplete')
  })

  it('creates privacy-safe defaults', () => {
    const profile = defaultProfile()
    expect(profile.fingerprint.webrtcPolicy).toBe('proxy-only')
    expect(profile.proxy.url).toBe('direct://')
  })
})
