import { useCallback, useEffect, useState } from 'react'
import { formatDateTime } from '../i18n/format'
import {
  portableProfileAPI,
  type PortableTemplate,
  type PortableTemplateUpdateRequest,
} from '../portableProfiles'

type TemplateDraft = {
  name: string
  profileName: string
  group: string
  notes: string
  tags: string
}

const emptyDraft: TemplateDraft = { name: '', profileName: '', group: '', notes: '', tags: '' }

export function TemplateMaintenanceWorkspace() {
  const nativeMode = portableProfileAPI.isNative()
  const [templates, setTemplates] = useState<PortableTemplate[]>([])
  const [editingID, setEditingID] = useState('')
  const [draft, setDraft] = useState<TemplateDraft>(emptyDraft)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const load = useCallback(async () => {
    if (!nativeMode) return
    setLoading(true)
    setError('')
    try { setTemplates(await portableProfileAPI.templates()) }
    catch (reason) { setError(errorText(reason)) }
    finally { setLoading(false) }
  }, [nativeMode])

  useEffect(() => { void load() }, [load])

  const beginEdit = (template: PortableTemplate) => {
    setEditingID(template.id)
    setDraft({
      name: template.name,
      profileName: template.payload.name,
      group: template.payload.group || '',
      notes: template.payload.notes || '',
      tags: template.payload.tags?.join(', ') || '',
    })
    setError('')
    setNotice('')
  }

  const cancelEdit = () => {
    setEditingID('')
    setDraft(emptyDraft)
    setError('')
  }

  const save = async (template: PortableTemplate) => {
    const request: PortableTemplateUpdateRequest = {
      templateId: template.id,
      name: draft.name.trim(),
      profileName: draft.profileName.trim(),
      group: draft.group.trim(),
      notes: draft.notes.trim(),
      tags: splitTags(draft.tags),
    }
    if (!request.name) { setError('模板名称不能为空。'); return }
    if (!request.profileName) { setError('默认环境名称不能为空。'); return }

    setSaving(true)
    setError('')
    setNotice('')
    try {
      const updated = await portableProfileAPI.updateTemplate(request)
      setTemplates((current) => current.map((item) => item.id === updated.id ? updated : item))
      setEditingID('')
      setDraft(emptyDraft)
      setNotice(`已更新模板“${updated.name}”，依赖要求和身份安全边界保持不变。`)
    } catch (reason) { setError(errorText(reason)) }
    finally { setSaving(false) }
  }

  return <section className="panel recovery-section">
    <div className="panel-heading">
      <div><span className="eyebrow">私有环境模板目录</span><h2>模板维护</h2><p>编辑可复用且不包含秘密的名称与默认值。浏览器内核、代理组件、路由、凭据要求和指纹设置保持不变；可复用种子仍然不会保存。</p></div>
      <button className="button secondary" disabled={!nativeMode || loading || saving} onClick={() => void load()}>{loading ? '正在刷新…' : '刷新模板'}</button>
    </div>
    {!nativeMode && <div className="form-error">模板维护需要 Wails 桌面运行时。</div>}
    {error && <div className="form-error">{error}</div>}
    {notice && <div className="info-banner"><strong>操作完成</strong><p>{notice}</p></div>}
    {templates.length === 0 ? <div className="lifecycle-empty">当前没有可编辑的便携环境模板。</div> : <div className="recovery-card-grid">
      {templates.map((template) => {
        const editing = editingID === template.id
        return <article className="recovery-card" key={template.id}>
          <div className="recovery-card-head"><strong>{template.name}</strong><span className="lifecycle-operation-status passed">始终创建新身份</span></div>
          <code>{template.id}</code>
          {!editing ? <>
            <dl>
              <div><dt>默认环境名称</dt><dd>{template.payload.name}</dd></div>
              <div><dt>分组</dt><dd>{template.payload.group || '无'}</dd></div>
              <div><dt>标签</dt><dd>{template.payload.tags?.join('、') || '无'}</dd></div>
              <div><dt>更新时间</dt><dd>{formatDateTime(template.updatedAt)}</dd></div>
            </dl>
            <button className="button secondary" disabled={!nativeMode || saving || Boolean(editingID)} onClick={() => beginEdit(template)}>编辑不含秘密的默认值</button>
          </> : <>
            <label>模板名称<input value={draft.name} maxLength={120} onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))} /></label>
            <label>默认环境名称<input value={draft.profileName} maxLength={120} onChange={(event) => setDraft((current) => ({ ...current, profileName: event.target.value }))} /></label>
            <label>分组<input value={draft.group} maxLength={120} onChange={(event) => setDraft((current) => ({ ...current, group: event.target.value }))} /></label>
            <label>标签<input value={draft.tags} placeholder="研究, 工作, QA" onChange={(event) => setDraft((current) => ({ ...current, tags: event.target.value }))} /></label>
            <label>备注<textarea value={draft.notes} maxLength={16384} onChange={(event) => setDraft((current) => ({ ...current, notes: event.target.value }))} /></label>
            <div className="info-banner"><strong>以下内容会自动保留</strong><p>与提供方兼容的指纹默认值、浏览器内核与代理组件身份、路由默认值和本机凭据要求不会在这里被修改。</p></div>
            <div className="recovery-card-actions">
              <button className="button primary" disabled={saving || !draft.name.trim() || !draft.profileName.trim()} onClick={() => void save(template)}>{saving ? '正在保存…' : '保存修改'}</button>
              <button className="button secondary" disabled={saving} onClick={cancelEdit}>取消</button>
            </div>
          </>}
        </article>
      })}
    </div>}
  </section>
}

function splitTags(value: string): string[] {
  const seen = new Set<string>()
  const result: string[] = []
  for (const raw of value.split(/[;,\n]/)) {
    const tag = raw.trim()
    const key = tag.toLowerCase()
    if (!tag || seen.has(key)) continue
    seen.add(key)
    result.push(tag)
  }
  return result
}

function errorText(reason: unknown): string {
  return reason instanceof Error ? reason.message : String(reason)
}
