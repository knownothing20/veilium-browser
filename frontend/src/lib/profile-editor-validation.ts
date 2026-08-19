import type { AdapterRecord, Profile } from '../types'

export type ProfileEditorValidationCode =
  | 'name-required'
  | 'kernel-required'
  | 'proxy-required'
  | 'adapter-required'

export function requiredAdapterKind(raw?: string): AdapterRecord['kind'] | undefined {
  const scheme = (raw || '').split(':', 1)[0].trim().toLowerCase()
  if (['vmess', 'vless', 'trojan', 'ss', 'shadowsocks'].includes(scheme)) return 'xray'
  if (['hysteria2', 'tuic', 'anytls'].includes(scheme)) return 'sing-box'
  return undefined
}

export function validateProfileEditorDraft(profile: Profile): ProfileEditorValidationCode | undefined {
  if (!profile.name.trim()) return 'name-required'
  if (!profile.kernel.id && !profile.kernel.executable.trim()) return 'kernel-required'
  if (!(profile.proxy.url || '').trim()) return 'proxy-required'
  if (requiredAdapterKind(profile.proxy.url) && !profile.proxy.adapterRef?.trim()) return 'adapter-required'
  return undefined
}

export function applyExplicitSystemProxy(profile: Profile, proxyUrl: string): Profile {
  const normalized = proxyUrl.trim()
  if (!normalized) throw new Error('system proxy URL is required')
  return {
    ...profile,
    proxy: { url: normalized },
  }
}
