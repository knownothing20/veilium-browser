import { useEffect, useState } from 'react'
import { formatDateTime } from '../i18n/format'
import type { CredentialRecord, CredentialSaveRequest } from '../types'
import { AppIcon } from './AppIcon'

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
    if (!window.confirm(`确定从 ${provider} 删除凭据“${record.name}”吗？`)) return
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
        <div className="panel-heading"><div><h2>{request.id ? '更新凭据' : '添加代理凭据'}</h2><p>密码会直接发送到 {provider}，不会返回到网页界面，也不会写入环境元数据。</p></div></div>
        <form onSubmit={submit}>
          <label>显示名称<input required value={request.name} onChange={(event) => setRequest((current) => ({ ...current, name: event.target.value }))} placeholder="例如：美国住宅代理" /></label>
          <label>用户名<input required autoComplete="off" value={request.username} onChange={(event) => setRequest((current) => ({ ...current, username: event.target.value }))} /></label>
          <label>密码<input required={!request.id} type="password" autoComplete="new-password" value={request.secret || ''} onChange={(event) => setRequest((current) => ({ ...current, secret: event.target.value }))} placeholder={request.id ? '留空则保留当前密码' : '仅保存在操作系统凭据存储中'} /></label>
          <div className="credential-form-actions">
            {request.id && <button type="button" className="button secondary" onClick={() => setRequest(emptyRequest)}>取消编辑</button>}
            <button className="button primary" disabled={!nativeMode || busy}>{busy ? '正在保存…' : request.id ? '更新凭据' : '保存凭据'}</button>
          </div>
        </form>
        {!nativeMode && <div className="info-banner credential-info"><strong>需要桌面运行时</strong><p>浏览器预览模式不会接收或保存密码。</p></div>}
        {error && <div className="form-error">{error}</div>}
      </section>

      <section className="credential-records">
        {records.length === 0
          ? <div className="panel empty-state"><div className="empty-icon"><AppIcon name="credential" size={25} /></div><h3>还没有保存凭据引用</h3><p>请通过桌面应用添加代理用户名和密码。</p></div>
          : records.map((record) => <article className="credential-card" key={record.id}>
            <div className="credential-card-head"><div className="credential-symbol"><AppIcon name="credential" /></div><div><h2>{record.name}</h2><code>{record.id}</code></div></div>
            <dl><div><dt>用户名</dt><dd>{record.username}</dd></div><div><dt>存储提供方</dt><dd>{provider}</dd></div><div><dt>更新时间</dt><dd>{formatDateTime(record.updatedAt)}</dd></div></dl>
            <div className="credential-card-note"><span>●</span><p>密码保存在 Veilium 元数据之外，无法从此页面查看或导出。</p></div>
            <div className="credential-card-actions"><button className="button secondary" disabled={busy || !nativeMode} onClick={() => edit(record)}>编辑名称与用户名</button><button className="button secondary danger-text" disabled={busy || !nativeMode} onClick={() => void remove(record)}>删除</button></div>
          </article>)}
      </section>
    </div>
  )
}
