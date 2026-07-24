import { useEffect, useMemo, useState } from 'react'
import {
  applyKernel,
  capabilityAllowsConfiguration,
  capabilityFor,
  capabilitiesFor,
  defaultProfile,
  providerTrust,
} from '../lib/model'
import { ui } from '../i18n'
import type {
  AdapterRecord,
  CapabilityID,
  CredentialRecord,
  KernelRecord,
  Profile,
  ProviderDescriptor,
} from '../types'

function requiredAdapterKind(raw?: string): AdapterRecord['kind'] | undefined {
  const scheme = (raw || '').split(':', 1)[0].trim().toLowerCase()
  if (['vmess', 'vless', 'trojan', 'ss', 'shadowsocks'].includes(scheme)) return 'xray'
  if (['hysteria2', 'tuic', 'anytls'].includes(scheme)) return 'sing-box'
  return undefined
}

function preferredProvider(providers: ProviderDescriptor[]): ProviderDescriptor | undefined {
  return providers.find((item) => item.id === 'custom-chromium')
    || providers.find((item) => item.id === 'native-chromium')
    || providers[0]
}

export function ProfileEditor({
  open,
  profile,
  providers,
  kernels,
  adapters,
  credentials,
  onClose,
  onSave,
}: {
  open: boolean
  profile?: Profile
  providers: ProviderDescriptor[]
  kernels: KernelRecord[]
  adapters: AdapterRecord[]
  credentials: CredentialRecord[]
  onClose: () => void
  onSave: (profile: Profile) => Promise<void>
}) {
  const initialProvider = preferredProvider(providers)
  const [draft, setDraft] = useState<Profile>(() => profile ? structuredClone(profile) : defaultProfile(initialProvider))
  const [tags, setTags] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    const next = profile ? structuredClone(profile) : defaultProfile(preferredProvider(providers))
    setDraft(next)
    setTags((next.tags || []).join(', '))
    setError('')
  }, [profile, providers, open])

  const selectedProvider = providers.find((item) => item.id === draft.kernel.provider) || preferredProvider(providers)
  const providerCapabilities = useMemo(
    () => capabilitiesFor(providers, draft.kernel.provider, draft.kernel.version),
    [providers, draft.kernel.provider, draft.kernel.version],
  )
  const trust = providerTrust(providers, draft.kernel.provider, draft.kernel.version)
  const providerBlocked = trust === 'disabled' || trust === 'invalid'
  const verifiedKernels = kernels.filter((item) => item.status === 'verified')
  const adapterKind = requiredAdapterKind(draft.proxy.url)
  const compatibleAdapters = adapters.filter((item) => item.status === 'verified' && item.kind === adapterKind)

  if (!open) return null

  const declaration = (id: CapabilityID) => capabilityFor(
    providers,
    draft.kernel.provider,
    draft.kernel.version,
    id,
  )
  const canConfigure = (id: CapabilityID) => capabilityAllowsConfiguration(declaration(id)?.status || 'unsupported')
  const update = <K extends keyof Profile>(key: K, value: Profile[K]) => setDraft((current) => ({ ...current, [key]: value }))
  const updateFingerprint = (key: keyof Profile['fingerprint'], value: string | number) => setDraft((current) => ({
    ...current,
    fingerprint: { ...current.fingerprint, [key]: value },
  }))
  const updateKernel = (key: keyof Profile['kernel'], value: string) => setDraft((current) => ({
    ...current,
    kernel: { ...current.kernel, [key]: value },
  }))
  const updateProxy = (key: keyof Profile['proxy'], value: string) => setDraft((current) => ({
    ...current,
    proxy: { ...current.proxy, [key]: value },
  }))

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (providerBlocked) {
      setError(ui.editor.blockedError(draft.kernel.provider, trust))
      return
    }
    setSaving(true)
    setError('')
    try {
      await onSave({
        ...draft,
        tags: tags.split(',').map((item) => item.trim()).filter(Boolean),
      })
      onClose()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setSaving(false)
    }
  }

  const trustDetail = trust === 'reviewed'
    ? ui.editor.reviewedTrust
    : trust === 'custom'
      ? ui.editor.customTrust
      : trust === 'legacy'
        ? ui.editor.legacyTrust
        : ui.editor.blockedTrust

  return (
    <div className="overlay" onMouseDown={onClose}>
      <form className="editor-panel guided-editor" onSubmit={submit} onMouseDown={(event) => event.stopPropagation()}>
        <header className="editor-header">
          <div>
            <span className="eyebrow">{ui.editor.eyebrow}</span>
            <h2>{profile ? ui.editor.editTitle : ui.editor.createTitle}</h2>
          </div>
          <button type="button" className="close-button" onClick={onClose} aria-label={ui.common.cancel}>×</button>
        </header>

        <div className="editor-scroll">
          <FormSection index="01" title={ui.editor.basic} detail={ui.editor.basicDetail}>
            <div className="form-grid two">
              <label>{ui.editor.name}<input required value={draft.name} onChange={(event) => update('name', event.target.value)} /></label>
              <label>{ui.editor.group}<input value={draft.group || ''} onChange={(event) => update('group', event.target.value)} /></label>
            </div>
            <label>{ui.editor.tags}<input value={tags} onChange={(event) => setTags(event.target.value)} /><small>{ui.editor.tagsHint}</small></label>
            <label>{ui.editor.notes}<textarea value={draft.notes || ''} onChange={(event) => update('notes', event.target.value)} /></label>
          </FormSection>

          <FormSection index="02" title={ui.editor.browser} detail={ui.editor.browserDetail}>
            <label>
              {ui.editor.registeredKernel}
              <select
                value={draft.kernel.id || ''}
                onChange={(event) => {
                  const record = verifiedKernels.find((item) => item.id === event.target.value)
                  if (record) setDraft((current) => applyKernel(current, record))
                  else setDraft((current) => ({ ...current, kernel: { ...current.kernel, id: undefined } }))
                }}
              >
                <option value="">{ui.editor.legacyExecutable}</option>
                {verifiedKernels.map((item) => <option key={item.id} value={item.id}>{item.name} · Chromium {item.version.split('.')[0]}</option>)}
              </select>
              <small>{ui.editor.kernelHint}</small>
            </label>

            <div className={`info-banner trust-banner ${providerBlocked ? 'danger' : ''}`}>
              <strong>{ui.editor.providerTrust}：{trustLabel(trust)}</strong>
              <p>{trustDetail}</p>
            </div>

            <details className="advanced-settings">
              <summary>{ui.editor.advancedBrowser}</summary>
              <div className="advanced-settings-content">
                <div className="form-grid two">
                  <label>
                    {ui.editor.provider}
                    <select
                      disabled={Boolean(draft.kernel.id)}
                      value={draft.kernel.provider}
                      onChange={(event) => {
                        const provider = providers.find((item) => item.id === event.target.value) || preferredProvider(providers)
                        const defaults = defaultProfile(provider)
                        setDraft((current) => ({ ...current, kernel: defaults.kernel, fingerprint: defaults.fingerprint }))
                      }}
                    >
                      {providers.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}
                    </select>
                  </label>
                  <label>
                    {ui.editor.chromiumVersion}
                    <select disabled={Boolean(draft.kernel.id)} value={draft.kernel.version} onChange={(event) => updateKernel('version', event.target.value)}>
                      {selectedProvider?.versions.map((version) => <option key={version}>{version}</option>)}
                    </select>
                  </label>
                </div>
                <label>{ui.editor.executablePath}<input required readOnly={Boolean(draft.kernel.id)} value={draft.kernel.executable} onChange={(event) => updateKernel('executable', event.target.value)} /></label>
                <div className="capability-strip">
                  {([
                    ['surface-seed', '稳定种子'],
                    ['surface-controls', '表面控制'],
                    ['custom-gpu', '自定义 GPU'],
                    ['hardware-concurrency', 'CPU 覆盖'],
                  ] as const).map(([id, label]) => {
                    const item = declaration(id)
                    return <span key={id} className={capabilityAllowsConfiguration(item?.status || 'unsupported') ? 'on' : ''} title={item?.limitation || '没有提供方声明'}>{label}：{capabilityStatusLabel(item?.status || 'unsupported')}</span>
                  })}
                </div>
                {providerCapabilities?.limitations?.length ? <small>{providerCapabilities.limitations.join(' · ')}</small> : null}
              </div>
            </details>
          </FormSection>

          <FormSection index="03" title={ui.editor.identity} detail={ui.editor.identityDetail}>
            <div className="form-grid three">
              <label>{ui.editor.platform}<select value={draft.fingerprint.platform} disabled={!canConfigure('platform')} onChange={(event) => updateFingerprint('platform', event.target.value)}><option value="windows">Windows</option><option value="linux">Linux</option><option value="macos">macOS</option></select></label>
              <label>{ui.editor.brand}<select value={draft.fingerprint.brand} disabled={!canConfigure('browser-brand')} onChange={(event) => updateFingerprint('brand', event.target.value)}><option>Chromium</option><option>Chrome</option><option>Edge</option><option>Opera</option><option>Vivaldi</option></select></label>
              <label>{ui.editor.language}<input value={draft.fingerprint.language} onChange={(event) => updateFingerprint('language', event.target.value)} /></label>
              <label>{ui.editor.timezone}<input value={draft.fingerprint.timezone} disabled={!canConfigure('timezone')} onChange={(event) => updateFingerprint('timezone', event.target.value)} /></label>
              <label>{ui.editor.screenWidth}<input type="number" min="800" max="7680" value={draft.fingerprint.screenWidth} onChange={(event) => updateFingerprint('screenWidth', Number(event.target.value))} /></label>
              <label>{ui.editor.screenHeight}<input type="number" min="600" max="4320" value={draft.fingerprint.screenHeight} onChange={(event) => updateFingerprint('screenHeight', Number(event.target.value))} /></label>
              <label>{ui.editor.cpuThreads}<input type="number" min="2" max="128" disabled={!canConfigure('hardware-concurrency')} value={draft.fingerprint.hardwareConcurrency || 0} onChange={(event) => updateFingerprint('hardwareConcurrency', Number(event.target.value))} /></label>
              <label>{ui.editor.webrtc}<select value={draft.fingerprint.webrtcPolicy} onChange={(event) => updateFingerprint('webrtcPolicy', event.target.value)}><option value="proxy-only">仅代理</option><option value="disabled">禁用</option><option value="default">默认</option></select></label>
              <label>{ui.editor.gpuProfile}<select value={draft.fingerprint.gpuProfile} onChange={(event) => updateFingerprint('gpuProfile', event.target.value)}><option value="auto">自动保持一致</option><option value="native">使用本机</option>{canConfigure('custom-gpu') && <option value="custom">自定义元数据</option>}</select></label>
            </div>
          </FormSection>

          <FormSection index="04" title={ui.editor.network} detail={ui.editor.networkDetail}>
            <label>{ui.editor.proxyUrl}<input value={draft.proxy.url || ''} onChange={(event) => updateProxy('url', event.target.value)} placeholder="direct://、http://proxy.example:8080 或 vless://…" /></label>
            <label>
              {ui.editor.credential}
              <select value={draft.proxy.credentialRef || ''} onChange={(event) => updateProxy('credentialRef', event.target.value)}>
                <option value="">{ui.editor.noCredential}</option>
                {credentials.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.username}</option>)}
              </select>
            </label>
            {adapterKind && <label>
              {ui.editor.managedAdapter}（{adapterKind}）
              <select required value={draft.proxy.adapterRef || ''} onChange={(event) => updateProxy('adapterRef', event.target.value)}>
                <option value="">{ui.editor.selectVerifiedAdapter}</option>
                {compatibleAdapters.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.version}</option>)}
              </select>
              <small>{ui.editor.adapterHint}</small>
            </label>}
            {!adapterKind && draft.proxy.adapterRef && <div className="info-banner danger"><strong>{ui.editor.adapterRejected}</strong><p>{ui.editor.adapterRejectedDetail}</p></div>}
          </FormSection>

          <FormSection index="05" title={ui.editor.review} detail={ui.editor.reviewDetail}>
            <div className="review-card">
              <strong>{draft.name || ui.editor.createTitle}</strong>
              <dl>
                <div><dt>{ui.editor.registeredKernel}</dt><dd>{verifiedKernels.find((item) => item.id === draft.kernel.id)?.name || ui.editor.legacyExecutable}</dd></div>
                <div><dt>{ui.editor.proxyUrl}</dt><dd>{draft.proxy.url || 'direct://'}</dd></div>
                <div><dt>{ui.editor.language}</dt><dd>{draft.fingerprint.language}</dd></div>
                <div><dt>{ui.editor.timezone}</dt><dd>{draft.fingerprint.timezone}</dd></div>
              </dl>
              <p>{ui.editor.reviewReady}</p>
            </div>
          </FormSection>
        </div>

        {error && <div className="form-error">{error}</div>}
        <footer className="editor-footer">
          <button type="button" className="button secondary" onClick={onClose}>{ui.common.cancel}</button>
          <button className="button primary" disabled={saving || providerBlocked}>{saving ? ui.common.saving : profile ? ui.editor.saveChanges : ui.editor.create}</button>
        </footer>
      </form>
    </div>
  )
}

function FormSection({ index, title, detail, children }: { index: string; title: string; detail: string; children: React.ReactNode }) {
  return <section className="form-section"><div className="section-heading"><span>{index}</span><div><h3>{title}</h3><p>{detail}</p></div></div>{children}</section>
}

function trustLabel(value: string): string {
  if (value === 'reviewed') return '已审查'
  if (value === 'custom') return '自定义'
  if (value === 'legacy') return '兼容模式'
  if (value === 'disabled') return '已禁用'
  if (value === 'invalid') return '无效'
  return value
}

function capabilityStatusLabel(value: string): string {
  if (value === 'supported') return '支持'
  if (value === 'unsupported') return '不支持'
  if (value === 'partial') return '有限支持'
  if (value === 'verified') return '已验证'
  return value
}
