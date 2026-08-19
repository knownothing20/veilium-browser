import { describe, expect, it } from 'vitest'
import { defaultProfile } from './model'
import { applyExplicitSystemProxy, requiredAdapterKind, validateProfileEditorDraft } from './profile-editor-validation'

describe('profile editor validation', () => {
  it('reports the first actionable missing field instead of relying on hidden browser validation', () => {
    const profile = defaultProfile()
    expect(validateProfileEditorDraft(profile)).toBe('name-required')
    profile.name = 'Example'
    expect(validateProfileEditorDraft(profile)).toBe('kernel-required')
    profile.kernel.executable = 'C:\\managed\\chrome.exe'
    profile.proxy.url = ''
    expect(validateProfileEditorDraft(profile)).toBe('proxy-required')
  })

  it('requires a matching adapter only for advanced routes', () => {
    const profile = defaultProfile()
    profile.name = 'Example'
    profile.kernel.executable = 'C:\\managed\\chrome.exe'
    profile.proxy.url = 'vless://proxy.example:443'
    expect(requiredAdapterKind(profile.proxy.url)).toBe('xray')
    expect(validateProfileEditorDraft(profile)).toBe('adapter-required')
    profile.proxy.adapterRef = 'adapter-1'
    expect(validateProfileEditorDraft(profile)).toBeUndefined()
  })

  it('applies an explicit system proxy without mutating identity and clears stale route references', () => {
    const profile = defaultProfile()
    profile.proxy = {
      url: 'vless://old.example:443',
      credentialRef: 'credential-old',
      adapterRef: 'adapter-old',
    }
    const identityBefore = structuredClone(profile.fingerprint)

    const updated = applyExplicitSystemProxy(profile, '  http://127.0.0.1:10100  ')

    expect(updated).not.toBe(profile)
    expect(updated.proxy).toEqual({ url: 'http://127.0.0.1:10100' })
    expect(updated.fingerprint).toEqual(identityBefore)
    expect(profile.proxy.credentialRef).toBe('credential-old')
    expect(profile.proxy.adapterRef).toBe('adapter-old')
  })

  it('rejects an empty system proxy result', () => {
    expect(() => applyExplicitSystemProxy(defaultProfile(), '  ')).toThrow('system proxy URL is required')
  })
})
