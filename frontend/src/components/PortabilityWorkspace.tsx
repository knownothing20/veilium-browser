import { useEffect, useMemo, useState } from 'react'
import { formatDateTime } from '../i18n/format'
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
  const [templateProfileNames, setTemplateProfileNames] = useState<Record<string, string>>({})
  const [busy, setBusy] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const selectedProfile = data.profiles.find((item) => item.id === profileId)
  const credentialTemplateCount = templates.filter((item) => item.payload.credentialRequired).length

  const loadTemplates = async () => {
    try {
      const loaded = await portableProfileAPI.templates()
      setTemplates(loaded)
      setTemplateProfileNames((current) => {
        const next: Record<string, string> = {}
        for (const template of loaded) next[template.id] = current[template.id] || `${template.payload.name} 模板副本`
        return next
      })
    } catch (reason) { setError(errorText(reason)) }
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
    if (!selectedProfile) throw new Error('请选择要导出的浏览器环境。')
    const destination = await portableProfileAPI.pickExport(selectedProfile.name)
    if (!destination) return
    const result = await portableProfileAPI.exportProfile({ profileId: selectedProfile.id, destination, identityMode })
    setExportResult(result)
    setNotice(`环境定义已导出到 ${result.path}`)
  })

  const chooseImport = () => run('preview', async () => {
    const path = await portableProfileAPI.pickImport()
    if (!path) return
    const next = await portableProfileAPI.previewImport(path)
    setPreview(next)
    setImportName(`${next.artifact.payload.name} 导入副本`)
    setIdentityMode(next.artifact.payload.identityMode || 'new-identity')
    setKernelId(next.kernelMatches[0]?.id || '')
    setAdapterId(next.adapterMatches[0]?.id || '')
    setCredentialId('')
  })

  const importProfile = () => run('import', async () => {
    if (!preview) throw new Error('请先选择并预览一个便携环境定义。')
    const result = await portableProfileAPI.importProfile({
      path: preview.path,
      name: importName,
      identityMode,
      kernelId,
      adapterId,
      credentialId,
    })
    setNotice(`已创建环境“${result.profile.name}”。${result.warnings.join(' ')}`)
    setPreview(undefined)
    await onRefresh()
  })

  const createTemplate = () => run('template-create', async () => {
    if (!selectedProfile) throw new Error('请选择一个来源环境。')
    if (!templateName.trim()) throw new Error('请输入模板名称。')
    await portableProfileAPI.createTemplate({ profileId: selectedProfile.id, name: templateName.trim() })
    setTemplateName('')
    await loadTemplates()
    setNotice('模板已创建。模板不包含浏览器数据、凭据、绝对路径或可复用的指纹种子。')
  })

  const deleteTemplate = (template: PortableTemplate) => run(`template-delete-${template.id}`, async () => {
    if (!window.confirm(`确定删除模板“${template.name}”吗？`)) return
    await portableProfileAPI.deleteTemplate(template.id)
    await loadTemplates()
    setNotice(`已删除模板“${template.name}”。`)
  })

  const resolveTemplate = (template: PortableTemplate) => {
    const matchedKernel = data.kernels.find((item) => item.status === 'verified'
      && item.provider === template.payload.kernel.provider
      && item.version === template.payload.kernel.version
      && item.sha256.toLowerCase() === template.payload.kernel.sha256.toLowerCase()
      && item.sizeBytes === template.payload.kernel.sizeBytes)
    const matchedAdapter = template.payload.adapter ? data.adapters.find((item) => item.status === 'verified'
      && item.kind === template.payload.adapter?.kind
      && item.version === template.payload.adapter?.version
      && item.sha256.toLowerCase() === template.payload.adapter?.sha256.toLowerCase()
      && item.sizeBytes === template.payload.adapter?.sizeBytes) : undefined
    const selectedCredential = template.payload.credentialRequired
      ? data.credentials.find((item) => item.id === templateCredentialId)
      : undefined
    const missing: string[] = []
    if (!matchedKernel) missing.push('匹配且已验证的浏览器内核')
    if (template.payload.adapter && !matchedAdapter) missing.push('匹配且已验证的代理组件')
    if (template.payload.credentialRequired && !selectedCredential) missing.push('本机凭据')
    return { matchedKernel, matchedAdapter, selectedCredential, missing, ready: missing.length === 0 }
  }

  const applyTemplate = (template: PortableTemplate) => run(`template-apply-${template.id}`, async () => {
    const resolution = resolveTemplate(template)
    if (!resolution.matchedKernel) throw new Error('本机没有与此模板匹配的已验证浏览器内核。')
    if (template.payload.adapter && !resolution.matchedAdapter) throw new Error('本机没有与此模板匹配的已验证代理组件。')
    if (template.payload.credentialRequired && !resolution.selectedCredential) throw new Error('应用模板前请选择本机凭据。')
    const requestedName = (templateProfileNames[template.id] || '').trim()
    if (!requestedName) throw new Error('请输入新环境名称。')
    const result = await portableProfileAPI.applyTemplate({
      templateId: template.id,
      name: requestedName,
      kernelId: resolution.matchedKernel.id,
      adapterId: resolution.matchedAdapter?.id,
      credentialId: resolution.selectedCredential?.id,
    })
    setNotice(`已创建环境“${result.profile.name}”，并生成新的环境 ID、受管目录和指纹种子。`)
    await onRefresh()
  })

  const readiness = useMemo(() => {
    if (!preview) return false
    return preview.kernelMatches.some((item) => item.id === kernelId)
      && (!preview.artifact.payload.adapter || preview.adapterMatches.some((item) => item.id === adapterId))
      && (!preview.credentialRequired || data.credentials.some((item) => item.id === credentialId))
  }, [preview, kernelId, adapterId, credentialId, data.credentials])

  return <section className="panel recovery-section">
    <div className="panel-heading"><div><h2>便携环境定义与模板</h2><p>在不同安装之间移动经过验证且不包含秘密的配置，不复制浏览器数据、凭据、二进制文件、本机 ID、绝对路径或 Evidence。</p></div></div>
    {!nativeMode && <div className="form-error">便携环境操作需要 Wails 桌面运行时。</div>}
    {error && <div className="form-error">{error}</div>}
    {notice && <div className="info-banner"><strong>操作完成</strong><p>{notice}</p></div>}

    <div className="settings-grid">
      <article className="panel setting-card">
        <h2>导出单个环境定义</h2>
        <p>只导出配置与依赖要求。目标文件是本机私有 JSON，不包含浏览器数据和密码。</p>
        <label>来源环境<select value={profileId} onChange={(event) => setProfileId(event.target.value)}>{data.profiles.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
        <label>身份模式<select value={identityMode} onChange={(event) => setIdentityMode(event.target.value as IdentityMode)}><option value="new-identity">创建新身份（推荐）</option><option value="preserve-identity">保留身份（高级）</option></select></label>
        {identityMode === 'preserve-identity' && <div className="form-error">不要在多个设备或多个环境中同时运行同一个被保留的身份。</div>}
        <button className="button primary" disabled={!nativeMode || !selectedProfile || Boolean(busy)} onClick={() => void exportProfile()}>{busy === 'export' ? '正在导出…' : '选择目标并导出'}</button>
        {exportResult && <dl><div><dt>数据身份</dt><dd>{exportResult.payloadSha256.slice(0, 20)}…</dd></div><div><dt>已排除内容</dt><dd>{exportResult.exclusions.length} 类敏感数据</dd></div></dl>}
      </article>

      <article className="panel setting-card">
        <h2>导入便携环境定义</h2>
        <p>创建本机新环境前，先预览完整性、排除内容、依赖匹配、身份模式和警告。</p>
        <button className="button secondary" disabled={!nativeMode || Boolean(busy)} onClick={() => void chooseImport()}>{busy === 'preview' ? '正在读取…' : '选择便携文件'}</button>
        {preview && <>
          <div className="recovery-card-head"><strong>{preview.artifact.payload.name}</strong><span className={`lifecycle-operation-status ${preview.ready ? 'passed' : 'failed'}`}>{preview.ready ? '依赖已找到' : '需要映射依赖'}</span></div>
          <label>新环境名称<input value={importName} onChange={(event) => setImportName(event.target.value)} /></label>
          <label>身份模式<select value={identityMode} onChange={(event) => setIdentityMode(event.target.value as IdentityMode)}><option value="new-identity">创建新身份</option>{preview.artifact.payload.fingerprint.seed && <option value="preserve-identity">保留导出的身份</option>}</select></label>
          <label>已验证浏览器内核<select value={kernelId} onChange={(event) => setKernelId(event.target.value)}><option value="">选择匹配内核</option>{preview.kernelMatches.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.kind} {item.version}</option>)}</select></label>
          {preview.artifact.payload.adapter && <label>已验证代理组件<select value={adapterId} onChange={(event) => setAdapterId(event.target.value)}><option value="">选择匹配代理组件</option>{preview.adapterMatches.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.kind} {item.version}</option>)}</select></label>}
          {preview.credentialRequired && <label>本机凭据<select value={credentialId} onChange={(event) => setCredentialId(event.target.value)}><option value="">选择本机凭据</option>{data.credentials.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.username}</option>)}</select></label>}
          <ul className="plain-list">{(preview.warnings || []).map((item) => <li key={item}>{item}</li>)}</ul>
          <button className="button primary" disabled={!readiness || !importName.trim() || Boolean(busy)} onClick={() => void importProfile()}>{busy === 'import' ? '正在创建…' : '创建导入环境'}</button>
        </>}
      </article>
    </div>

    <div className="panel-heading"><div><h2>可复用模板</h2><p>模板保留不含秘密的默认配置和依赖要求，但不会保存可复用的指纹种子。</p></div></div>
    <div className="toolbar">
      <select value={profileId} onChange={(event) => setProfileId(event.target.value)}>{data.profiles.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select>
      <input value={templateName} onChange={(event) => setTemplateName(event.target.value)} placeholder="模板名称" />
      <button className="button primary" disabled={!nativeMode || !selectedProfile || !templateName.trim() || Boolean(busy)} onClick={() => void createTemplate()}>{busy === 'template-create' ? '正在创建…' : '从环境创建模板'}</button>
    </div>
    {credentialTemplateCount > 0 && <label>需要路由秘密的模板使用此本机凭据<select value={templateCredentialId} onChange={(event) => setTemplateCredentialId(event.target.value)}><option value="">应用时选择本机凭据</option>{data.credentials.map((item) => <option key={item.id} value={item.id}>{item.name} · {item.username}</option>)}</select></label>}
    <div className="recovery-card-grid">
      {templates.length === 0 ? <div className="lifecycle-empty">还没有创建便携模板。</div> : templates.map((template) => {
        const resolution = resolveTemplate(template)
        const requestedName = templateProfileNames[template.id] || ''
        return <article className="recovery-card" key={template.id}>
          <div className="recovery-card-head"><strong>{template.name}</strong><span className={`lifecycle-operation-status ${resolution.ready ? 'passed' : 'failed'}`}>{resolution.ready ? '可以应用' : '需要映射依赖'}</span></div>
          <code>{template.id}</code>
          <dl><div><dt>环境默认名称</dt><dd>{template.payload.name}</dd></div><div><dt>内核要求</dt><dd>{template.payload.kernel.provider} · {template.payload.kernel.version}</dd></div><div><dt>网络路由</dt><dd>{template.payload.proxyUrl || '直连'}</dd></div><div><dt>凭据</dt><dd>{template.payload.credentialRequired ? '必须明确选择本机凭据' : '不需要'}</dd></div></dl>
          {!resolution.ready && <div className="form-error">缺少：{resolution.missing.join('、')}。</div>}
          <label>新环境名称<input value={requestedName} onChange={(event) => setTemplateProfileNames((current) => ({ ...current, [template.id]: event.target.value }))} /></label>
          <details>
            <summary>查看不含秘密的模板详情</summary>
            <dl>
              <div><dt>分组</dt><dd>{template.payload.group || '无'}</dd></div>
              <div><dt>标签</dt><dd>{template.payload.tags?.join('、') || '无'}</dd></div>
              <div><dt>备注</dt><dd>{template.payload.notes || '无'}</dd></div>
              <div><dt>指纹种子</dt><dd>不保存，应用模板时重新生成</dd></div>
              <div><dt>内核 SHA-256</dt><dd><code>{template.payload.kernel.sha256}</code></dd></div>
              {template.payload.adapter && <div><dt>代理组件要求</dt><dd>{template.payload.adapter.kind} · {template.payload.adapter.version}</dd></div>}
              <div><dt>创建时间</dt><dd>{formatDateTime(template.createdAt)}</dd></div>
              <div><dt>更新时间</dt><dd>{formatDateTime(template.updatedAt)}</dd></div>
            </dl>
          </details>
          <div className="recovery-card-actions"><button className="button primary" disabled={!nativeMode || Boolean(busy) || !resolution.ready || !requestedName.trim()} onClick={() => void applyTemplate(template)}>{busy === `template-apply-${template.id}` ? '正在应用…' : '创建环境'}</button><button className="button secondary danger-text" disabled={!nativeMode || Boolean(busy)} onClick={() => void deleteTemplate(template)}>{busy === `template-delete-${template.id}` ? '正在删除…' : '删除模板'}</button></div>
        </article>
      })}
    </div>
  </section>
}

function errorText(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }
