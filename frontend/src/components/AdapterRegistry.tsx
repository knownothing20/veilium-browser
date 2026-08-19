import { useMemo, useState } from 'react'
import { formatBytes, statusLabel } from '../i18n/format'
import type {
  AdapterImportRequest,
  AdapterInstallRequest,
  AdapterRecord,
  AdapterReleasePin,
  AdapterValidationReport,
} from '../types'
import { AppIcon } from './AppIcon'

const defaults: Record<AdapterRecord['kind'], Pick<AdapterImportRequest, 'licenseSpdx' | 'sourceUrl'>> = {
  xray: { licenseSpdx: 'MPL-2.0', sourceUrl: 'https://github.com/XTLS/Xray-core' },
  'sing-box': { licenseSpdx: 'GPL-3.0-or-later', sourceUrl: 'https://github.com/SagerNet/sing-box' },
}

export function AdapterRegistry({
  records,
  pins,
  reports,
  runtimePlatform,
  runtimeArch,
  nativeMode,
  busy,
  error,
  onPick,
  onImport,
  onInstall,
  onVerify,
  onValidate,
  onDelete,
}: {
  records: AdapterRecord[]
  pins: AdapterReleasePin[]
  reports: Record<string, AdapterValidationReport>
  runtimePlatform: string
  runtimeArch: string
  nativeMode: boolean
  busy: boolean
  error?: string
  onPick: () => Promise<string>
  onImport: (request: AdapterImportRequest) => Promise<void>
  onInstall: (request: AdapterInstallRequest) => Promise<void>
  onVerify: (record: AdapterRecord) => Promise<void>
  onValidate: (record: AdapterRecord) => Promise<void>
  onDelete: (record: AdapterRecord) => Promise<void>
}) {
  const [request, setRequest] = useState<AdapterImportRequest>({
    name: '',
    kind: 'xray',
    version: pins.find((pin) => pin.kind === 'xray')?.version ?? '',
    sourcePath: '',
    ...defaults.xray,
  })
  const [accepted, setAccepted] = useState<Record<string, boolean>>({})

  const protocols = useMemo(
    () => request.kind === 'xray' ? ['VMess', 'VLESS', 'Trojan', 'Shadowsocks'] : ['Hysteria2', 'TUIC', 'AnyTLS'],
    [request.kind],
  )
  const releases = useMemo(() => {
    const unique = new Map<string, AdapterReleasePin>()
    for (const pin of pins) unique.set(`${pin.kind}@${pin.version}`, pin)
    return Array.from(unique.values())
  }, [pins])

  async function pick() {
    const path = await onPick()
    if (path) {
      setRequest((current) => ({
        ...current,
        sourcePath: path,
        name: current.name || path.split(/[\\/]/).pop() || current.kind,
      }))
    }
  }

  async function submit() {
    await onImport(request)
    setRequest((current) => ({ ...current, name: '', sourcePath: '' }))
  }

  function changeKind(kind: AdapterRecord['kind']) {
    const pin = pins.find((item) => item.kind === kind)
    setRequest((current) => ({ ...current, kind, version: pin?.version ?? '', ...defaults[kind] }))
  }

  return <>
    <section className="panel official-release-panel">
      <div className="panel-heading">
        <div><h2>固定版本安装器</h2><p>不会自动下载。每次安装都需要明确接受许可证，并且只使用仓库内固定的资源、大小和 SHA-256。</p></div>
      </div>
      <div className="official-pin-grid">
        {releases.map((release) => {
          const key = `${release.kind}@${release.version}`
          const pin = pins.find((item) => item.kind === release.kind && item.version === release.version && item.platform === runtimePlatform && item.arch === runtimeArch)
          const installed = records.find((record) => record.official && record.kind === release.kind && record.version === release.version && record.officialPlatform === runtimePlatform && record.officialArch === runtimeArch && record.status === 'verified')
          return <article key={key}>
            <div className="official-pin-title">
              <div><strong>{release.kind === 'xray' ? 'Xray' : 'sing-box'}</strong><span>{release.tag}</span></div>
              <span className={installed ? 'official-installed' : 'official-available'}>{installed ? '已安装' : pin ? '可安装' : '当前平台不可用'}</span>
            </div>
            <code>{release.repository}</code>
            <small>{pin ? `${pin.assetName} · ${formatBytes(pin.archiveSizeBytes)}` : `没有适用于 ${runtimePlatform}/${runtimeArch} 的固定资源`}</small>
            <small>许可证：{release.licenseSpdx}</small>
            <label className="license-acknowledgement">
              <input type="checkbox" checked={Boolean(accepted[key])} disabled={!pin || Boolean(installed) || busy} onChange={(event) => setAccepted((current) => ({ ...current, [key]: event.target.checked }))} />
              <span>我已了解 {release.licenseSpdx} 许可证，并明确请求下载此精确版本。</span>
            </label>
            <button className="button primary official-install-button" disabled={!nativeMode || busy || !pin || Boolean(installed) || !accepted[key]} onClick={() => void onInstall({ kind: release.kind, version: release.version, licenseAccepted: true })}>
              {installed ? '已安装' : busy ? '正在处理…' : '下载、验证并安装'}
            </button>
          </article>
        })}
      </div>
    </section>

    <section className="panel adapter-import">
      <div className="panel-heading"><div><h2>导入本机代理组件</h2><p>自定义版本仍可本机导入。Veilium 会计算可执行文件哈希，只有与内置固定资源的摘要和大小完全一致时才授予官方身份。</p></div></div>
      <div className="adapter-import-grid">
        <label>名称<input value={request.name} onChange={(event) => setRequest((current) => ({ ...current, name: event.target.value }))} placeholder="官方或自定义运行组件" /></label>
        <label>组件类型<select value={request.kind} onChange={(event) => changeKind(event.target.value as AdapterRecord['kind'])}><option value="xray">Xray</option><option value="sing-box">sing-box</option></select></label>
        <label>版本<input value={request.version} onChange={(event) => setRequest((current) => ({ ...current, version: event.target.value }))} placeholder="上游声明版本" /></label>
        <label>SPDX 许可证<input value={request.licenseSpdx} onChange={(event) => setRequest((current) => ({ ...current, licenseSpdx: event.target.value }))} /></label>
        <label className="adapter-source">来源地址<input value={request.sourceUrl} onChange={(event) => setRequest((current) => ({ ...current, sourceUrl: event.target.value }))} /></label>
        <label className="adapter-path">可执行文件路径<div className="path-picker"><input readOnly value={request.sourcePath} placeholder="选择本机代理组件可执行文件…" /><button type="button" className="button secondary" onClick={() => void pick()} disabled={!nativeMode || busy}>选择文件</button></div></label>
        <button className="button primary adapter-import-button" onClick={() => void submit()} disabled={!nativeMode || busy || !request.name.trim() || !request.version.trim() || !request.sourcePath.trim() || !request.licenseSpdx.trim() || !request.sourceUrl.trim()}>{busy ? '正在处理…' : '导入并识别'}</button>
      </div>
      <div className="adapter-capabilities"><span>已审查协议范围</span>{protocols.map((protocol) => <strong key={protocol}>{protocol}</strong>)}</div>
      {!nativeMode && <div className="info-banner"><strong>需要桌面运行时</strong><p>浏览器预览模式不能下载、读取或执行本机二进制文件。</p></div>}
      {error && <div className="form-error">{error}</div>}
    </section>

    <div className="adapter-list">
      {records.length === 0
        ? <section className="panel empty-state"><div className="empty-icon"><AppIcon name="network" size={25} /></div><h3>还没有受管代理组件</h3><p>为高级代理协议分配组件前，请安装固定版本或导入本机二进制文件。</p></section>
        : records.map((record) => {
          const report = reports[record.id]
          return <article className="adapter-card" key={record.id}>
            <div className="adapter-card-head">
              <div className="adapter-logo">{record.kind === 'xray' ? 'X' : 'S'}</div>
              <div><h2>{record.name}</h2><code>{record.kind} · {record.version}</code></div>
              <div className="adapter-status-stack"><span className={`kernel-status ${record.status}`}>{statusLabel(record.status)}</span><span className={record.official ? 'official-badge' : 'custom-badge'}>{record.official ? `官方 ${record.officialTag}` : '本机自定义'}</span></div>
            </div>
            <div className="adapter-protocols">{(record.protocols || []).map((protocol) => <span key={protocol}>{protocol}</span>)}</div>
            <dl>
              <div><dt>可执行文件 SHA</dt><dd title={record.sha256}>{record.sha256.slice(0, 16)}…{record.sha256.slice(-8)}</dd></div>
              <div><dt>官方身份</dt><dd>{record.official ? `${record.officialAsset} · ${record.officialPlatform}/${record.officialArch}` : '未匹配内置官方资源'}</dd></div>
              <div><dt>许可证</dt><dd>{record.licenseSpdx}</dd></div>
              <div><dt>来源</dt><dd title={record.sourceUrl}>{record.sourceUrl}</dd></div>
              <div><dt>受管路径</dt><dd title={record.executable}>{record.executable}</dd></div>
            </dl>
            {report && <div className="official-validation-report"><div><strong>官方配置检查已通过</strong><span>{report.versionText.split('\n')[0]}</span></div><ul>{(report.checks || []).map((check) => <li key={check.id}>✓ {check.label}</li>)}</ul></div>}
            <div className="kernel-actions">
              <button className="button secondary" disabled={busy} onClick={() => void onVerify(record)}>完整性检查</button>
              <button className="button secondary" disabled={busy || !nativeMode || !record.official || record.status !== 'verified'} title={record.official ? '运行官方二进制配置检查' : '只有与固定资源精确匹配的官方组件才能执行此检查'} onClick={() => void onValidate(record)}>官方配置检查</button>
              <button className="button secondary danger-text" disabled={busy} onClick={() => void onDelete(record)}>移除</button>
            </div>
          </article>
        })}
    </div>
  </>
}
