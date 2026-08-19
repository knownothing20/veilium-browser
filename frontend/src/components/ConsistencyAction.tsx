import { useState } from 'react'
import { formatDateTime } from '../i18n/format'
import type { Profile } from '../types'

type HealthStatus = 'healthy' | 'degraded' | 'blocked' | 'unknown'
type CheckStatus = 'passed' | 'warning' | 'failed' | 'unknown'

interface ConsistencyCheck {
  id: string
  status: CheckStatus
  expected?: string
  observed?: string
  reasonCode?: string
  detail?: string
}

interface ConsistencyResult {
  schemaVersion: number
  rulesRevision: string
  profileId: string
  inputDigest: string
  evidenceRunId?: string
  evidenceFresh: boolean
  status: HealthStatus
  window: { width: number; height: number; deviceScaleFactor: number; source: 'explicit' | 'legacy-screen-fallback' }
  checks: ConsistencyCheck[]
  blockingReasons?: string[]
  degradedReasons?: string[]
  generatedAt: string
}

type ConsistencyAPI = {
  ProfileConsistency: (profileId: string) => Promise<ConsistencyResult>
  UpdateProfile: (profile: Profile) => Promise<Profile>
}

function api(): ConsistencyAPI | undefined {
  const value = window as Window & { go?: { main?: { DesktopApp?: ConsistencyAPI } } }
  return value.go?.main?.DesktopApp
}

export function ConsistencyAction({ profile, nativeMode }: { profile: Profile; nativeMode: boolean }) {
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<ConsistencyResult>()
  const [windowWidth, setWindowWidth] = useState(profile.fingerprint.windowWidth || 0)
  const [windowHeight, setWindowHeight] = useState(profile.fingerprint.windowHeight || 0)
  const [deviceScaleFactor, setDeviceScaleFactor] = useState(profile.fingerprint.deviceScaleFactor || 0)

  async function inspect() {
    setOpen(true)
    setLoading(true)
    setError('')
    setWindowWidth(profile.fingerprint.windowWidth || 0)
    setWindowHeight(profile.fingerprint.windowHeight || 0)
    setDeviceScaleFactor(profile.fingerprint.deviceScaleFactor || 0)
    try {
      const desktop = api()
      if (!desktop) throw new Error('环境一致性检查仅在桌面应用中可用。')
      setResult(await desktop.ProfileConsistency(profile.id))
    } catch (reason) {
      setResult(undefined)
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally { setLoading(false) }
  }

  async function saveWindowPlan() {
    const desktop = api()
    if (!desktop) { setError('环境一致性检查仅在桌面应用中可用。'); return }
    setSaving(true)
    setError('')
    try {
      const next: Profile = { ...profile, fingerprint: { ...profile.fingerprint, windowWidth, windowHeight, deviceScaleFactor } }
      await desktop.UpdateProfile(next)
      setResult(await desktop.ProfileConsistency(profile.id))
    } catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { setSaving(false) }
  }

  function useLegacyFallback() { setWindowWidth(0); setWindowHeight(0); setDeviceScaleFactor(0) }

  return <>
    <button className="button compact secondary" title={nativeMode ? '检查身份与窗口一致性' : '需要桌面运行时'} disabled={!nativeMode || !profile.kernel.id} onClick={() => void inspect()}>一致性</button>
    {open && <div className="overlay" onMouseDown={() => setOpen(false)}>
      <section className="evidence-dialog" onMouseDown={(event) => event.stopPropagation()}>
        <header className="editor-header">
          <div><span className="eyebrow">基于真实证据的环境健康</span><h2>{profile.name} · 一致性检查</h2></div>
          <button type="button" className="close-button" onClick={() => setOpen(false)} aria-label="关闭">×</button>
        </header>
        <div className="editor-scroll evidence-report-scroll">
          {loading && <div className="info-banner"><strong>正在评估当前环境…</strong></div>}
          {error && <div className="form-error">{error}</div>}
          {result && <>
            <div className="evidence-summary-grid">
              <div><span>状态</span><strong className={`status-pill ${result.status}`}>{healthStatusLabel(result.status)}</strong></div>
              <div><span>窗口</span><strong>{result.window.width} × {result.window.height}</strong></div>
              <div><span>缩放比例</span><strong>{result.window.deviceScaleFactor}</strong></div>
              <div><span>浏览器证据</span><strong>{result.evidenceFresh ? '有效' : '缺失或已过期'}</strong></div>
            </div>
            <div className="info-banner"><strong>{result.window.source === 'explicit' ? '明确窗口方案' : '旧版屏幕参数兼容模式'}</strong><p>规则 {result.rulesRevision} · 生成于 {formatDateTime(result.generatedAt)}</p></div>
            <section className="form-section">
              <div className="section-heading"><span>W</span><div><h3>受管窗口方案</h3><p>明确设置窗口尺寸可保持每次启动稳定。三个值全部设为 0 时，继续使用现有屏幕尺寸兼容方案。</p></div></div>
              <div className="form-grid three">
                <label>窗口宽度<input type="number" min="0" max="16384" value={windowWidth} onChange={(event) => setWindowWidth(Number(event.target.value))} /></label>
                <label>窗口高度<input type="number" min="0" max="16384" value={windowHeight} onChange={(event) => setWindowHeight(Number(event.target.value))} /></label>
                <label>设备缩放比例<input type="number" min="0" max="8" step="0.05" value={deviceScaleFactor} onChange={(event) => setDeviceScaleFactor(Number(event.target.value))} /></label>
              </div>
              <button type="button" className="text-button" onClick={useLegacyFallback}>使用旧版屏幕参数兼容模式</button>
            </section>
            {(result.blockingReasons?.length || 0) > 0 && <div className="form-error"><strong>阻止启动的原因</strong><p>{result.blockingReasons!.join(' · ')}</p></div>}
            {(result.degradedReasons?.length || 0) > 0 && <div className="info-banner"><strong>受限或未知原因</strong><p>{result.degradedReasons!.join(' · ')}</p></div>}
            <div className="evidence-observation-list">
              {result.checks.map((check) => <article key={check.id} className={`evidence-observation ${check.status}`}>
                <div className="evidence-observation-head"><strong>{check.id}</strong><span>{checkStatusLabel(check.status)}</span></div>
                {(check.expected || check.observed) && <dl>{check.expected && <div><dt>预期</dt><dd>{check.expected}</dd></div>}{check.observed && <div><dt>实际</dt><dd>{check.observed}</dd></div>}</dl>}
                {(check.detail || check.reasonCode) && <p>{check.detail || check.reasonCode}</p>}
              </article>)}
            </div>
          </>}
        </div>
        <footer className="editor-footer">
          <button type="button" className="button secondary" onClick={() => void inspect()} disabled={loading || saving}>重新检查</button>
          <button type="button" className="button secondary" onClick={() => void saveWindowPlan()} disabled={loading || saving}>{saving ? '正在保存…' : '保存窗口方案'}</button>
          <button type="button" className="button primary" onClick={() => setOpen(false)}>关闭</button>
        </footer>
      </section>
    </div>}
  </>
}

function healthStatusLabel(value: HealthStatus): string { const labels: Record<HealthStatus, string> = { healthy: '正常', degraded: '受限', blocked: '已阻止', unknown: '未知' }; return labels[value] }
function checkStatusLabel(value: CheckStatus): string { const labels: Record<CheckStatus, string> = { passed: '通过', warning: '警告', failed: '失败', unknown: '未知' }; return labels[value] }
