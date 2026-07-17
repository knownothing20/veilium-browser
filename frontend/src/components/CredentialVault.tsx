import { useEffect, useState } from 'react'
import type { CredentialRecord, CredentialSaveRequest } from '../types'

const emptyRequest: CredentialSaveRequest = { name: '', username: '', secret: '' }

export function CredentialVault({
  records,
  provider,
  nativeMode,
  onSave,
  onDelete,
}: {
  records: CredentialRecord[]
  provider: string
  nativeMode: boolean
  onSave: (request: CredentialSaveRequest) => Promise<void>
  onDelete: (id: string) => Promise<void>
}) {
  const [request, setRequest] = useState<CredentialSaveRequest>(emptyRequest)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (request.id && !records.some((record) => record.id === request.id)) setRequest(emptyRequest)
  }, [records, request.id])

  function edit(record: CredentialRecord) {
    setRequest({ id: record.id, name: record.name, username: record.username, secret: '' })
    setError('')
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    setBusy(true)
    setError('')
    try {
      await onSave(request)
      setRequest(emptyRequest)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setBusy(false)
    }
  }

  async function remove(record: CredentialRecord) {
    if (!window.confirm(`Delete “${record.name}” from ${provider}?`)) return
    setBusy(true)
    setError('')
    try {
      await onDelete(record.id)
      if (request.id === record.id) setRequest(emptyRequest)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="credential-layout">
      <section className="panel credential-form-card">
        <div className="panel-heading"><div><h2>{request.id ? 'Update credential' : 'Add proxy credential'}</h2><p>The password is sent directly to {provider} and is never returned to the web interface.</p></div></div>
        <form onSubmit={submit}>
          <label>Display name<input required value={request.name} onChange={(event) => setRequest((current) => ({ ...current, name: event.target.value }))} placeholder="US residential proxy" /></label>
          <label>Username<input required autoComplete="off" value={request.username} onChange={(event) => setRequest((current) => ({ ...current, username: event.target.value }))} /></label>
          <label>Password<input required={!request.id} type="password" autoComplete="new-password" value={request.secret || ''} onChange={(event) => setRequest((current) => ({ ...current, secret: event.target.value }))} placeholder={request.id ? 'Leave blank to keep the current password' : 'Stored only in the OS vault'} /></label>
          <div className="credential-form-actions">
            {request.id && <button type="button" className="button secondary" onClick={() => setRequest(emptyRequest)}>Cancel edit</button>}
            <button className="button primary" disabled={!nativeMode || busy}>{busy ? 'Saving…' : request.id ? 'Update credential' : 'Store credential'}</button>
          </div>
        </form>
        {!nativeMode && <div className="info-banner credential-info"><strong>Desktop runtime required</strong><p>Browser preview mode never accepts or stores passwords.</p></div>}
        {error && <div className="form-error">{error}</div>}
      </section>

      <section className="credential-records">
        {records.length === 0
          ? <div className="panel empty-state"><div className="empty-icon">◇</div><h3>No stored credential references</h3><p>Add a proxy username and password through the desktop application.</p></div>
          : records.map((record) => <article className="credential-card" key={record.id}>
            <div className="credential-card-head"><div className="credential-symbol">◇</div><div><h2>{record.name}</h2><code>{record.id}</code></div></div>
            <dl><div><dt>Username</dt><dd>{record.username}</dd></div><div><dt>Provider</dt><dd>{provider}</dd></div><div><dt>Updated</dt><dd>{new Date(record.updatedAt).toLocaleString()}</dd></div></dl>
            <div className="credential-card-note"><span>●</span><p>Password stored outside Veilium metadata. It cannot be revealed from this screen.</p></div>
            <div className="credential-card-actions"><button className="button secondary" disabled={busy || !nativeMode} onClick={() => edit(record)}>Edit metadata</button><button className="button secondary danger-text" disabled={busy || !nativeMode} onClick={() => void remove(record)}>Delete</button></div>
          </article>)}
      </section>
    </div>
  )
}
