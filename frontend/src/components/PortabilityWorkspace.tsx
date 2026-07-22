import { useEffect, useMemo, useState } from 'react'
import type { RecoveryWorkspaceData } from '../localRecovery'
import {
  portableProfileAPI,
  type IdentityMode,
  type PortableExportResult,
  type PortableImportPreview,
  type PortableTemplate,
} from '../portableProfiles'

export function PortabilityWorkspace({ data, onRefresh }: { data: RecoveryWorkspaceData; onRefresh: () => Promise<void> }) {
  const nativeMode = portableProfileAPI.isNative()
  const [profileId, setProfileId] = useState(data.profiles[0]?.id || '')
  const [identityMode, setIdentityMode] = useState<IdentityMode>('new-identity')
  const [exportResult, setExportResult] = useState<PortableExportResult>()
  const [preview, setPreview] = useState<PortableImportPreview>()
  const [importName, setImportName] = useState('')
  const [kernelId, setKernelId] = useState('')
  const [adapterId, setAdapterId] = useState('')
  const [credentialId, setCredentialId] = useState('')
  const [templateCredentialId, setTemplateCredentialId] = useState('')
  const [templates, setTemplates] = useState<PortableTemplate[]>([])
  const [templateName, setTemplateName] = useState('')
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const selectedProfile = data.profiles.find((item) => item.id === profileId)
  const credentialTemplateCount = templates.filter((item) => item.payload.credentialRequired).length

  const loadTemplates = async () => {
    try { setTemplates(await portableProfileAPI.templates()) }
    catch (reason) { setError(errorText(reason)) }
  }

  useEffect(() => { void loadTemplates() }, [nativeMode])
  useEffect(() => {
    if (!profileId && data.profiles[0]) setProfileId(data.profiles[0].id)
  }, [data.profiles, profileId])
  useEffect(() => {
    if (templateCredentialId && !data.credentials.some((item) => item.id === templateCredentialId)) setTemplateCredentialId('')
  }, [data.credentials, templateCredentialId])

  const run = async (key: string, action: () => Promise<void>) => {
    setBusy(key)
    setError('')
    setNotice('')
    try { await action() }
    catch (reason) { setError(errorText(reason)) }
    finally { setBusy('') }
  }

  const exportProfile = () => run('export', async () => {
    if (!selectedProfile) throw new Error('Select a Profile to export.')
    const destination = await portableProfileAPI.pickExport(selectedProfile.name)
    if (!destination) return
    const result = await portableProfileAPI.exportProfile({ profileId: selectedProfile.id, destination, identityMode })
    setExportResult(result)
    setNotice(`Portable Profile exported to ${result.path}`)
  })

  const chooseImport = () => run('preview', async () => {
    const path = await portableProfileAPI.pickImport()
    if (!path) return
    const next = await portableProfileAPI.previewImport(path)
    setPreview(next)
    setImportName(`${next.artifact.payload.name} Imported`)
    setIdentityMode(next.artifact.payload.identityMode || 'new-identity')
    setKernelId(next.kernelMatches[0]?.id || '')
    setAdapterId(next.adapterMatches[0]?.id || '')
    setCredentialId('')
  })

  const importProfile = () => run('import', async () => {
    if (!preview) throw new Error('Choose and preview a portable Profile first.')
    const result = await portableProfileAPI.importProfile({
      path: preview.path,
      name: importName,
      identityMode,
      kernelId,
      adapterId,
      credentialId,
    })
    setNotice(`Created Profile “${result.profile.name}”. ${result.warnings.join(' ')}`)
    setPreview(undefined)
    await onRefresh()
  })

  const createTemplate = () => run('template-create', async () => {
    if (!selectedProfile) throw new Error('Select a source Profile.')
    if (!templateName.trim()) throw new Error('Enter a template name.')
    await portableProfileAPI.createTemplate({ profileId: selectedProfile.id, name: templateName.trim() })
    setTemplateName('')
    await loadTemplates()
    setNotice('Template created without browser data, credentials, local paths, or a reusable fingerprint seed.')
  })

  const deleteTemplate = (template: PortableTemplate) => run(`template-delete-${template.id}`, async () => {
    if (!window.confirm(`Delete template “${template.name}”?`)) return
    await portableProfileAPI.deleteTemplate(template.id)
    await loadTemplates()
  })

  const applyTemplate = (template: PortableTemplate) => run(`template-apply-${template.id}`, async () => {
    const matchedKernel = data.kernels.find((item) => item.status === 'verified'
      && item.provider === template.payload.kernel.provider
      && item.version === template.payload.kernel.version
      && item.sha256.toLowerCase() === template.payload.kernel.sha256.toLowerCase()
      && item.sizeBytes === template.payload.kernel.sizeBytes)
    if (!matchedKernel) throw new Error('No verified local Kernel matches this template.')
    const matchedAdapter = template.payload.adapter ? data.adapters.find((item) => item.status === 'verified'
      && item.kind === template.payload.adapter?.kind
      && item.version === template.payload.adapter?.version
      && item.sha256.toLowerCase() === template.payload.adapter?.sha256.toLowerCase()
      && item.sizeBytes === template.payload.adapter?.sizeBytes) : undefined
    if (template.payload.adapter && !matchedAdapter) throw new Error('No verified local proxy adapter matches this template.')
    const selectedCredential = template.payload.credentialRequired
      ? data.credentials.find((item) => item.id === templateCredentialId)
      : undefined
    if (template.payload.credentialRequired && !selectedCredential) throw new Error('Select a local vault credential before applying this template.')
    const result = await portableProfileAPI.applyTemplate({
      templateId: template.id,
      name: `${template.payload.name} from template`,
      kernelId: matchedKernel.id,
      adapterId: matchedAdapter?.id,
      credentialId: selectedCredential?.id,
    })
    setNotice(`Created Profile “${result.profile.name}” with a new ID, managed directory, and seed.`)
    await onRefresh()
  })

  const readiness = useMemo(() => {
    if (!preview) return false
    return preview.kernelMatches.some((item) => item.id === kernelId)
      && (!preview.artifact.payload.adapter || preview.adapterMatches.some((item) => item.id === adapterId))
      && (!preview.credentialRequired || data.credentials.some((item) => item.id === credentialId))
  }, [preview, kernelId, adapterId, credentialId, data.credentials])

  return <section className="panel recovery-section">
    <div className="panel-heading"><div><h2>Portable Profiles and templates</h2><p>Move validated non-secret configuration between installations without copying browser data, credentials, binaries, local IDs, local paths, or Evidence.</p></div></div>
    {!nativeMode && <div className="form-error">Portable Profile actions require the Wails desktop runtime.</div>}
    {error && <div className="form-error">{error}</div>}
    {notice && <div className="info-banner"><strong>Completed</strong><p>{notice}</p></div>}

    <div className="settings-grid">
      <article className="panel setting-card">
        <h2>Export one Profile</h2>
        <p>Export configuration and dependency requirements only. The destination file is private local JSON.</p>
        <label>Source Profile<select value={profileId} onChange={(event) => setProfileId(event.target.value)}>{data.profiles.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
        <label>Identity transfer<select value={identityMode} onChange={(event) => setIdentityMode(event.target.value as IdentityMode)}><option value="new-identity">New identity (recommended)</option><option value="preserve-identity">Preserve identity (advanced)</option></select></label>
        {identityMode === 'preserve-identity' && <div className="form-error">Do not run a preserved identity simultaneously on multiple devices or Profiles.</div>}
        <button className="button primary" disabled={!nativeMode || !selectedProfile || Boolean(busy)} onClick={() => void exportProfile()}>{busy === 'export' ? 'Exporting…' : 'Choose destination and export'}</button>
        {exportResult && <dl><div><dt>Payload identity</dt><dd>{exportResult.payloadSha256.slice(0, 20)}…</dd></div><div><dt>Excluded</dt><dd>{exportResult.exclusions.length} sensitive categories</dd></div></dl>}
      </article>

      <article className="panel setting-card">
        <h2>Import portable Profile</h2>
        <p>Preview integrity, exclusions, dependency matches, identity mode, and warnings before creating a new local Profile.</p>
        <button className="button secondary" disabled={!nativeMode || Boolean(busy)} onClick={() => void chooseImport()}>{busy === 'preview' ? 'Reading…' : 'Choose portable file'}</button>
        {preview && <>
          <div className="recovery-card-head"><strong>{preview.artifact.payload.name}</strong><span className={`lifecycle-operation-status ${preview.ready ? 'passed' : 'failed'}`}>{preview.ready ? 'dependencies found' : 'mapping required'}</span></div>
          <label>New Profile name<input value={importName} onChange={(event) => setImportName(event.target.value)} /></label>
          <label>Identity mode<select value={identityMode} onChange={(event) => setIdentityMode(event.target.value as IdentityMode)}><option value="new-identity">New identity</option>{preview.artifact.payload.fingerprint.seed && <option value="preserve-identity">Preserve exported identity</option>}</select></label>
          <label>Verified Kernel<select value={kernelId} onChange={(event) => setKernelId(event.target.value)}><option value="">Select matching Kernel</option>{preview.kernelMatches.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.kind} {item.version}</option>)}</select></label>
          {preview.artifact.payload.adapter && <label>Verified adapter<select value={adapterId} onChange={(event) => setAdapterId(event.target.value)}><option value="">Select matching adapter</option>{preview.adapterMatches.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.kind} {item.version}</option>)}</select></label>}
          {preview.credentialRequired && <label>Local vault credential<select value={credentialId} onChange={(event) => setCredentialId(event.target.value)}><option value="">Select local credential</option>{data.credentials.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.username}</option>)}</select></label>}
          <ul className="plain-list">{preview.warnings.map((item) => <li key={item}>{item}</li>)}</ul>
          <button className="button primary" disabled={!readiness || Boolean(busy)} onClick={() => void importProfile()}>{busy === 'import' ? 'Creating…' : 'Create imported Profile'}</button>
        </>}
      </article>
    </div>

    <div className="panel-heading"><div><h2>Reusable templates</h2><p>Templates keep non-secret defaults and dependency requirements, but never retain a reusable fingerprint seed.</p></div></div>
    <div className="toolbar">
      <select value={profileId} onChange={(event) => setProfileId(event.target.value)}>{data.profiles.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select>
      <input value={templateName} onChange={(event) => setTemplateName(event.target.value)} placeholder="Template name" />
      <button className="button primary" disabled={!nativeMode || !selectedProfile || !templateName.trim() || Boolean(busy)} onClick={() => void createTemplate()}>{busy === 'template-create' ? 'Creating…' : 'Create from Profile'}</button>
    </div>
    {credentialTemplateCount > 0 && <label>Credential for templates that require a route secret<select value={templateCredentialId} onChange={(event) => setTemplateCredentialId(event.target.value)}><option value="">Select local credential when applying</option>{data.credentials.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.username}</option>)}</select></label>}
    <div className="recovery-card-grid">
      {templates.length === 0 ? <div className="lifecycle-empty">No portable templates have been created.</div> : templates.map((template) => <article className="recovery-card" key={template.id}>
        <div className="recovery-card-head"><strong>{template.name}</strong><span className="lifecycle-operation-status passed">new identity</span></div>
        <code>{template.id}</code>
        <dl><div><dt>Profile defaults</dt><dd>{template.payload.name}</dd></div><div><dt>Kernel requirement</dt><dd>{template.payload.kernel.provider} · {template.payload.kernel.version}</dd></div><div><dt>Route</dt><dd>{template.payload.proxyUrl || 'Direct'}</dd></div><div><dt>Credential</dt><dd>{template.payload.credentialRequired ? 'Explicit local selection required' : 'Not required'}</dd></div></dl>
        <div className="recovery-card-actions"><button className="button primary" disabled={!nativeMode || Boolean(busy) || (template.payload.credentialRequired && !templateCredentialId)} onClick={() => void applyTemplate(template)}>{busy === `template-apply-${template.id}` ? 'Applying…' : 'Create Profile'}</button><button className="button secondary danger-text" disabled={!nativeMode || Boolean(busy)} onClick={() => void deleteTemplate(template)}>{busy === `template-delete-${template.id}` ? 'Deleting…' : 'Delete'}</button></div>
      </article>)}
    </div>
  </section>
}

function errorText(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }
