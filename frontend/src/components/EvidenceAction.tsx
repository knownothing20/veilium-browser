import { useMemo, useState } from 'react'
import { formatDateTime } from '../i18n/format'
import { backend } from '../lib/backend'
import { evidenceStatusClass, evidenceStatusLabel, evidenceSummary, latestEvidence } from '../lib/evidence'
import type { EvidenceRun, Profile, RuntimeSession } from '../types'

export function EvidenceAction({ profile, session, nativeMode }: { profile: Profile; session?: RuntimeSession; nativeMode: boolean }) {
  const [open, setOpen] = useState(false)
  const [runs, setRuns] = useState<EvidenceRun[]>([])
  const [selectedID, setSelectedID] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const selected = useMemo(() => runs.find((run) => run.id === selectedID) || latestEvidence(runs), [runs, selectedID])
  const canReview = nativeMode && Boolean(profile.kernel.id)
  const ready = canReview && session?.state === 'ready' && Boolean(session.cdpPort)

  async function load(clearError = true) {
    if (clearError) setError('')
    try {
      const items = await backend.listEvidence(profile.id)
      setRuns(items)
      if (!selectedID && items[0]) setSelectedID(items[0].id)
    } catch (reason) { setError(errorText(reason)) }
  }
  async function show() { setOpen(true); await load() }
  async function runEvidence() {
    setBusy(true)
    setError('')
    try {
      const result = await backend.runEvidence(profile.id)
      await load(false)
      setSelectedID(result.id)
    } catch (reason) {
      await load(false)
      setError(errorText(reason))
    } finally { setBusy(false) }
  }
  async function cancel() { try { await backend.cancelEvidence(profile.id) } catch (reason) { setError(errorText(reason)) } }
  async function remove(run: EvidenceRun) {
    if (!window.confirm('确定删除此本机浏览器检测报告吗？')) return
    try {
      await backend.deleteEvidence(run.id)
      const next = runs.filter((item) => item.id !== run.id)
      setRuns(next)
      setSelectedID(next[0]?.id || '')
    } catch (reason) { setError(errorText(reason)) }
  }

  return <>
    <button className="button compact secondary" title={!canReview ? '需要受管桌面环境' : ready ? '运行或查看浏览器检测' : '可以查看历史报告；打开浏览器后才能收集新报告'} disabled={!canReview} onClick={() => void show()}>浏览器检测</button>
    {open && <div className="evidence-overlay" onMouseDown={() => setOpen(false)}>
      <section className="evidence-dialog" onMouseDown={(event) => event.stopPropagation()}>
        <header>
          <div><span className="eyebrow">受控的本机浏览器观测</span><h2>{profile.name} · 浏览器检测</h2><p>不会收集 Cookie、浏览内容、凭据或远程探测数据。</p></div>
          <button className="close-button" onClick={() => setOpen(false)} aria-label="关闭">×</button>
        </header>
        <div className="evidence-toolbar">
          <button className="button primary" disabled={!ready || busy} onClick={() => void runEvidence()}>{busy ? '正在收集…' : '运行浏览器检测'}</button>
          {busy && <button className="button secondary" onClick={() => void cancel()}>取消</button>}
          <button className="button secondary" disabled={busy} onClick={() => void load()}>刷新</button>
        </div>
        {!ready && <div className="info-banner"><strong>当前为查看模式</strong><p>打开此受管浏览器环境后，才能收集新的真实浏览器报告。</p></div>}
        {error && <div className="form-error">{error}</div>}
        <div className="evidence-layout">
          <aside className="evidence-run-list">
            {runs.length === 0 && <p className="evidence-empty">还没有本机浏览器检测报告。</p>}
            {runs.map((run) => <button className={selected?.id === run.id ? 'selected' : ''} key={run.id} onClick={() => setSelectedID(run.id)}><strong>{evidenceStatusLabel(run.status)}</strong><span>{formatDateTime(run.startedAt)}</span><small>{run.providerId} · {evidenceSummary(run)}</small></button>)}
          </aside>
          <div className="evidence-report">
            {!selected ? <p className="evidence-empty">请选择或运行一份浏览器检测报告。</p> : <>
              <div className="evidence-report-head">
                <div><span className={`status-pill ${evidenceStatusClass(selected.status)}`}>{evidenceStatusLabel(selected.status)}</span><h3>{evidenceSummary(selected)}</h3><p>{selected.providerId} 修订 {selected.providerRevision} · {trustLabel(selected.providerTrust)} · Chromium {selected.browserVersion}</p></div>
                <button className="button secondary danger-text" onClick={() => void remove(selected)}>删除报告</button>
              </div>
              {selected.failureDetail && <div className="evidence-failure"><strong>{selected.failureCode || 'collection-failed'}</strong><p>{selected.failureDetail}</p></div>}
              {selected.limitations?.length ? <ul className="evidence-limitations">{selected.limitations.map((item) => <li key={item}>{item}</li>)}</ul> : null}
              <div className="evidence-observations">
                {(selected.observations || []).map((observation) => <article key={`${observation.context}-${observation.id}`}>
                  <div><strong>{observation.id}</strong><span className={`observation-status ${observation.status}`}>{observationStatusLabel(observation.status)}</span></div>
                  <small>{contextLabel(observation.context)}{observation.capabilityId ? ` · ${observation.capabilityId}` : ''}</small>
                  {observation.expected && <p><b>预期：</b> {observation.expected}</p>}
                  {observation.observed && <p><b>实际：</b> {observation.observed}</p>}
                  {(observation.reasonCode || observation.detail) && <p className="observation-detail">{observation.reasonCode}{observation.detail ? ` — ${observation.detail}` : ''}</p>}
                </article>)}
              </div>
            </>}
          </div>
        </div>
      </section>
    </div>}
  </>
}

function observationStatusLabel(value: string): string { const labels: Record<string, string> = { passed: '通过', partial: '部分通过', failed: '失败', unavailable: '不可用', skipped: '已跳过' }; return labels[value] || value }
function contextLabel(value: string): string { const labels: Record<string, string> = { 'top-level': '顶层页面', iframe: '内嵌页面', worker: '后台 Worker' }; return labels[value] || value }
function trustLabel(value: string): string { const labels: Record<string, string> = { reviewed: '已审查', custom: '自定义', legacy: '兼容模式', disabled: '已禁用', invalid: '无效' }; return labels[value] || value }
function errorText(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }
