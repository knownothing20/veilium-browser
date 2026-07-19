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
        <h2>Reviewed official Chromium Snapshot</h2>
        <p>One fixed Windows x64 package. Veilium never resolves latest or silently updates it.</p>
      </div>
      <span className={`kernel-status ${installed ? 'verified' : available ? 'available' : 'missing'}`}>
        {installed ? 'installed' : available ? 'available' : 'unavailable'}
      </span>
    </div>

    <dl className="kernel-pin-grid">
      <div><dt>Chromium</dt><dd>{pin.browserVersion}</dd></div>
      <div><dt>Snapshot revision</dt><dd>{pin.snapshotRevision}</dd></div>
      <div><dt>Archive</dt><dd>{(pin.archiveSizeBytes / 1024 / 1024).toFixed(1)} MB</dd></div>
      <div><dt>Package identity</dt><dd>{pin.packageFileCount} files · {pin.packageTreeSha256.slice(0, 16)}…</dd></div>
    </dl>

    <div className="info-banner">
      <strong>Exact reviewed boundary</strong>
      <p>Reviewed status applies only to this revision, archive SHA-256, complete package tree, executable, Windows amd64, and accepted Evidence. Stock Chromium fingerprint overrides remain unsupported.</p>
    </div>

    <label className="checkbox-row kernel-license-row">
      <input
        type="checkbox"
        checked={accepted}
        onChange={(event) => onAcceptedChange(event.target.checked)}
        disabled={!available || installed}
      />
      <span>I acknowledge the Chromium BSD-3-Clause license, <code>chrome://credits/</code> third-party notices, snapshot limitations, and this explicit download.</span>
    </label>

    <div className="kernel-actions official-kernel-actions">
      <button
        className="button primary"
        disabled={disabled}
        onClick={() => onInstall({ providerId: pin.providerId, version: pin.browserVersion, licenseAccepted: accepted })}
      >
        {busy ? 'Downloading and verifying…' : installed ? 'Installed and verified' : available ? 'Download, verify, and install' : 'Windows x64 only'}
      </button>
    </div>

    <ul className="plain-list kernel-limitations">
      {pin.limitations.map((item) => <li key={item}>{item}</li>)}
    </ul>
  </section>
}
