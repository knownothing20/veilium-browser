import { useCallback, useEffect, useState } from 'react'
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
    try {
      setTemplates(await portableProfileAPI.templates())
    } catch (reason) {
      setError(errorText(reason))
    } finally {
      setLoading(false)
    }
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
    if (!request.name) {
      setError('Template name is required.')
      return
    }
    if (!request.profileName) {
      setError('Default Profile name is required.')
      return
    }

    setSaving(true)
    setError('')
    setNotice('')
    try {
      const updated = await portableProfileAPI.updateTemplate(request)
      setTemplates((current) => current.map((item) => item.id === updated.id ? updated : item))
      setEditingID('')
      setDraft(emptyDraft)
      setNotice(`Updated template “${updated.name}” without changing its dependency or identity-safety requirements.`)
    } catch (reason) {
      setError(errorText(reason))
    } finally {
      setSaving(false)
    }
  }

  return <section className="panel recovery-section">
    <div className="panel-heading">
      <div>
        <span className="eyebrow">M5.3 template catalog</span>
        <h2>Template maintenance</h2>
        <p>Edit reusable non-secret labels and defaults. Kernel, adapter, route, credential requirements, and fingerprint settings remain unchanged; reusable seeds remain excluded.</p>
      </div>
      <button className="button secondary" disabled={!nativeMode || loading || saving} onClick={() => void load()}>{loading ? 'Refreshing…' : 'Refresh templates'}</button>
    </div>
    {!nativeMode && <div className="form-error">Template maintenance requires the Wails desktop runtime.</div>}
    {error && <div className="form-error">{error}</div>}
    {notice && <div className="info-banner"><strong>Completed</strong><p>{notice}</p></div>}
    {templates.length === 0 ? <div className="lifecycle-empty">No portable templates are available to edit.</div> : <div className="recovery-card-grid">
      {templates.map((template) => {
        const editing = editingID === template.id
        return <article className="recovery-card" key={template.id}>
          <div className="recovery-card-head">
            <strong>{template.name}</strong>
            <span className="lifecycle-operation-status passed">new identity only</span>
          </div>
          <code>{template.id}</code>
          {!editing ? <>
            <dl>
              <div><dt>Default Profile</dt><dd>{template.payload.name}</dd></div>
              <div><dt>Group</dt><dd>{template.payload.group || 'None'}</dd></div>
              <div><dt>Tags</dt><dd>{template.payload.tags?.join(', ') || 'None'}</dd></div>
              <div><dt>Updated</dt><dd>{formatTime(template.updatedAt)}</dd></div>
            </dl>
            <button className="button secondary" disabled={!nativeMode || saving || Boolean(editingID)} onClick={() => beginEdit(template)}>Edit non-secret defaults</button>
          </> : <>
            <label>Template name<input value={draft.name} maxLength={120} onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))} /></label>
            <label>Default Profile name<input value={draft.profileName} maxLength={120} onChange={(event) => setDraft((current) => ({ ...current, profileName: event.target.value }))} /></label>
            <label>Group<input value={draft.group} maxLength={120} onChange={(event) => setDraft((current) => ({ ...current, group: event.target.value }))} /></label>
            <label>Tags<input value={draft.tags} placeholder="research, work, qa" onChange={(event) => setDraft((current) => ({ ...current, tags: event.target.value }))} /></label>
            <label>Notes<textarea value={draft.notes} maxLength={16384} onChange={(event) => setDraft((current) => ({ ...current, notes: event.target.value }))} /></label>
            <div className="info-banner"><strong>Preserved automatically</strong><p>Provider-compatible fingerprint defaults, Kernel and adapter identities, route defaults, and the local-credential requirement are not edited here.</p></div>
            <div className="recovery-card-actions">
              <button className="button primary" disabled={saving || !draft.name.trim() || !draft.profileName.trim()} onClick={() => void save(template)}>{saving ? 'Saving…' : 'Save changes'}</button>
              <button className="button secondary" disabled={saving} onClick={cancelEdit}>Cancel</button>
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

function formatTime(value: string): string {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function errorText(reason: unknown): string {
  return reason instanceof Error ? reason.message : String(reason)
}
