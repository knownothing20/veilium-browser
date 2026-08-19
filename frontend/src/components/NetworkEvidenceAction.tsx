import { useEffect, useMemo, useState } from 'react'
import { formatDateTime } from '../i18n/format'
import { networkBackend } from '../lib/network-backend'
import type { NetworkEvidenceRun, NetworkProbeDefinition, NetworkProbeSet } from '../network-types'
import type { Profile, RuntimeSession } from '../types'

const defaultSet = (): NetworkProbeSet => ({
  schemaVersion: 1,
  id: 'local-network-probes',
  revision: 1,
  definitions: [
    { schemaVersion: 1, id: 'exit', revision: 1, kind: 'exit-ip', httpsUrl: '', timeoutSeconds: 10, maxResponseBytes: 4096, selfHostable: true, privacyNote: '只返回受控浏览器请求所观察到的公网 IP。' },
    { schemaVersion: 1, id: 'stun', revision: 1, kind: 'webrtc-stun', stunServer: '', timeoutSeconds: 10, selfHostable: true, privacyNote: '只接收本次受控运行中受限的 STUN 交换。' },
    { schemaVersion: 1, id: 'dns', revision: 1, kind: 'delegated-dns', dnsZone: '', dnsResultUrl: '', timeoutSeconds: 10, maxResponseBytes: 4096, selfHostable: true, privacyNote: '只记录一次性委托 DNS 查询结果。' },
  ],
})

export function NetworkEvidenceAction({ profile, session, nativeMode }: { profile: Profile; session?: RuntimeSession; nativeMode: boolean }) {
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [configured, setConfigured] = useState(false)
  const [probeSet, setProbeSet] = useState<NetworkProbeSet>(defaultSet)
  const [runs, setRuns] = useState<NetworkEvidenceRun[]>([])
  const [matrixCount, setMatrixCount] = useState<number>()
  const ready = session?.state === 'ready' && Boolean(session.cdpPort)
  const latest = runs[0]

  useEffect(() => {
    if (!open || !nativeMode) return
    void refresh()
  }, [open, nativeMode, profile.id])

  const definitions = useMemo(() => ({
    exit: probeSet.definitions.find((item) => item.kind === 'exit-ip'),
    stun: probeSet.definitions.find((item) => item.kind === 'webrtc-stun'),
    dns: probeSet.definitions.find((item) => item.kind === 'delegated-dns'),
  }), [probeSet])

  async function refresh() {
    setBusy(true)
    setError('')
    try {
      const [configuration, reports] = await Promise.all([networkBackend.configured(), networkBackend.list(profile.id)])
      setConfigured(configuration.configured)
      setProbeSet(configuration.configured ? configuration.probeSet : defaultSet())
      setRuns(reports)
    } catch (reason) { setError(message(reason)) }
    finally { setBusy(false) }
  }

  function updateDefinition(kind: NetworkProbeDefinition['kind'], patch: Partial<NetworkProbeDefinition>) {
    setProbeSet((current) => ({ ...current, definitions: current.definitions.map((item) => item.kind === kind ? { ...item, ...patch } : item) }))
  }

  async function save() {
    setBusy(true); setError('')
    try { const saved = await networkBackend.saveProbeSet(probeSet); setProbeSet(saved); setConfigured(true) }
    catch (reason) { setError(message(reason)) }
    finally { setBusy(false) }
  }
  async function removeConfig() {
    setBusy(true); setError('')
    try { await networkBackend.deleteProbeSet(); setConfigured(false); setProbeSet(defaultSet()) }
    catch (reason) { setError(message(reason)) }
    finally { setBusy(false) }
  }
  async function run() {
    setBusy(true); setError('')
    try { await networkBackend.run(profile.id); setRuns(await networkBackend.list(profile.id)) }
    catch (reason) { setError(message(reason)); setRuns(await networkBackend.list(profile.id).catch(() => runs)) }
    finally { setBusy(false) }
  }
  async function deleteRun(id: string) {
    setBusy(true); setError('')
    try { await networkBackend.delete(id); setRuns(await networkBackend.list(profile.id)) }
    catch (reason) { setError(message(reason)) }
    finally { setBusy(false) }
  }
  async function loadMatrix() {
    setBusy(true); setError('')
    try { const matrix = await networkBackend.matrix(); setMatrixCount(matrix.entries.length) }
    catch (reason) { setError(message(reason)) }
    finally { setBusy(false) }
  }

  return <>
    <button className="button compact secondary" title={nativeMode ? '网络证据' : '需要桌面运行时'} disabled={!nativeMode} onClick={() => setOpen(true)}>网络证据</button>
    {open && <div className="overlay diagnostic-overlay" onMouseDown={() => setOpen(false)}>
      <aside className="diagnostic-drawer" onMouseDown={(event) => event.stopPropagation()}>
        <header className="editor-header">
          <div><span className="eyebrow">受控浏览器网络探针</span><h2>网络证据</h2><p>{profile.name}</p></div>
          <button className="close-button" onClick={() => setOpen(false)} aria-label="关闭">×</button>
        </header>
        <div className="diagnostic-content">
          {error && <div className="form-error">{error}</div>}
          <div className="diagnostic-limitations"><strong>明确配置探针</strong><p>Veilium 不包含隐藏的公共探针。请配置允许受控浏览器页面进行 CORS 请求的 HTTPS 或可自托管端点。</p></div>
          <label>探针集 ID<input value={probeSet.id} onChange={(event) => setProbeSet({ ...probeSet, id: event.target.value })} /></label>
          <label>修订号<input type="number" min={1} value={probeSet.revision} onChange={(event) => setProbeSet({ ...probeSet, revision: Number(event.target.value) || 1 })} /></label>
          <label>出口 IP HTTPS 地址<input value={definitions.exit?.httpsUrl || ''} placeholder="https://probe.example/ip" onChange={(event) => updateDefinition('exit-ip', { httpsUrl: event.target.value })} /></label>
          <label>STUN 服务器<input value={definitions.stun?.stunServer || ''} placeholder="stun:stun.example:3478" onChange={(event) => updateDefinition('webrtc-stun', { stunServer: event.target.value })} /></label>
          <label>委托 DNS 域<input value={definitions.dns?.dnsZone || ''} placeholder="probe.example" onChange={(event) => updateDefinition('delegated-dns', { dnsZone: event.target.value })} /></label>
          <label>DNS 结果地址<input value={definitions.dns?.dnsResultUrl || ''} placeholder="https://probe.example/dns-result" onChange={(event) => updateDefinition('delegated-dns', { dnsResultUrl: event.target.value })} /></label>
          <div className="diagnostic-actions">
            <button className="button secondary" disabled={busy} onClick={() => void save()}>{configured ? '更新探针集' : '保存探针集'}</button>
            {configured && <button className="button secondary" disabled={busy} onClick={() => void removeConfig()}>删除配置</button>}
            <button className="button primary" disabled={busy || !configured || !ready} title={!ready ? '请先打开环境并等待浏览器进入就绪状态' : ''} onClick={() => void run()}>{busy ? '正在运行…' : '运行网络证据'}</button>
          </div>
          {!ready && <div className="loading-block">受管浏览器会话必须正在运行并处于就绪状态。</div>}
          {latest && <>
            <div className="diagnostic-summary">
              <span className={`diagnostic-status ${latest.status}`}>{statusLabel(latest.status)}</span>
              <div><span>路由</span><strong>{latest.route.kind}</strong></div>
              <div><span>探针集</span><strong>{latest.probeSetId} r{latest.probeSetRevision}</strong></div>
              <div><span>过期时间</span><strong>{formatDateTime(latest.expiresAt)}</strong></div>
            </div>
            <div className="diagnostic-checks">
              {(latest.observations || []).map((item) => <article className={`diagnostic-check ${item.status}`} key={item.id}>
                <span className="diagnostic-check-icon">{icon(item.status)}</span>
                <div><div className="diagnostic-check-title"><strong>{probeKindLabel(item.probeKind)}</strong><span>{statusLabel(item.status)}</span></div><p>{item.detail || item.reasonCode || (item.values || []).join('、')}</p></div>
              </article>)}
            </div>
            {(latest.limitations || []).map((item) => <p key={item}>{item}</p>)}
            <div className="diagnostic-actions"><button className="button secondary danger-text" disabled={busy} onClick={() => void deleteRun(latest.id)}>删除最新报告</button></div>
          </>}
          <div className="diagnostic-actions"><button className="button secondary" disabled={busy} onClick={() => void loadMatrix()}>生成兼容性矩阵</button>{matrixCount !== undefined && <span>{matrixCount} 条精确组合记录</span>}</div>
        </div>
      </aside>
    </div>}
  </>
}

function message(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }
function icon(status: string): string { if (status === 'passed') return '✓'; if (status === 'failed') return '×'; if (status === 'partial') return '!'; return '–' }
function statusLabel(value: string): string { const labels: Record<string, string> = { passed: '通过', failed: '失败', partial: '部分完成', running: '进行中', cancelled: '已取消', unavailable: '不可用' }; return labels[value] || value }
function probeKindLabel(value: string): string { const labels: Record<string, string> = { 'exit-ip': '出口 IP', 'webrtc-stun': 'WebRTC STUN', 'delegated-dns': '委托 DNS' }; return labels[value] || value }
