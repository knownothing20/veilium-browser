import { formatBytes } from '../i18n/format'
import type { KernelInstallRequest, KernelRecord, KernelReleasePin } from '../types'

type Props = {
  pin?: KernelReleasePin
  records: KernelRecord[]
  runtimePlatform: string
  runtimeArch: string
  nativeMode: boolean
  busy: boolean
  accepted: boolean
  onAcceptedChange: (accepted: boolean) => void
  onInstall: (request: KernelInstallRequest) => void
}

export function OfficialKernelCard({
  pin,
  records,
  runtimePlatform,
  runtimeArch,
  nativeMode,
  busy,
  accepted,
  onAcceptedChange,
  onInstall,
}: Props) {
  if (!pin) return null

  const installed = records.some((item) =>
    item.provider === pin.providerId
    && item.version === pin.browserVersion
    && item.snapshotRevision === pin.snapshotRevision
    && item.packageTreeSha256 === pin.packageTreeSha256
    && item.status === 'verified',
  )
  const available = runtimePlatform === pin.platform && runtimeArch === pin.arch
  const disabled = busy || !nativeMode || !available || !accepted || installed

  return <section className="panel kernel-import official-kernel-card">
    <div className="panel-heading">
      <div>
        <h2>已审查的官方 Chromium 快照</h2>
        <p>固定的 Windows x64 浏览器包。Veilium 不会自动解析最新版，也不会静默更新。</p>
      </div>
      <span className={`kernel-status ${installed ? 'verified' : available ? 'available' : 'missing'}`}>
        {installed ? '已安装' : available ? '可安装' : '当前平台不可用'}
      </span>
    </div>

    <dl className="kernel-pin-grid">
      <div><dt>Chromium 版本</dt><dd>{pin.browserVersion}</dd></div>
      <div><dt>快照修订号</dt><dd>{pin.snapshotRevision}</dd></div>
      <div><dt>压缩包大小</dt><dd>{formatBytes(pin.archiveSizeBytes)}</dd></div>
      <div><dt>完整包身份</dt><dd>{pin.packageFileCount} 个文件 · {pin.packageTreeSha256.slice(0, 16)}…</dd></div>
    </dl>

    <div className="info-banner">
      <strong>精确的审查边界</strong>
      <p>“已审查”只适用于这个修订号、压缩包 SHA-256、完整目录树、可执行文件、Windows amd64 平台和已接受的真实运行证据。原版 Chromium 不支持的指纹覆盖不会因为安装此包而变成可用。</p>
    </div>

    <label className="checkbox-row kernel-license-row">
      <input
        type="checkbox"
        checked={accepted}
        onChange={(event) => onAcceptedChange(event.target.checked)}
        disabled={!available || installed}
      />
      <span>我已了解 Chromium BSD-3-Clause 许可证、<code>chrome://credits/</code> 第三方声明、快照限制，并明确同意下载此固定版本。</span>
    </label>

    <div className="kernel-actions official-kernel-actions">
      <button
        className="button primary"
        disabled={disabled}
        onClick={() => onInstall({ providerId: pin.providerId, version: pin.browserVersion, licenseAccepted: accepted })}
      >
        {busy ? '正在下载并验证…' : installed ? '已安装并通过验证' : available ? '下载、验证并安装' : '仅支持 Windows x64'}
      </button>
    </div>

    <ul className="plain-list kernel-limitations">
      {(pin.limitations || []).map((item) => <li key={item}>{item}</li>)}
    </ul>
  </section>
}
