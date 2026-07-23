import { useEffect, useMemo, useState } from 'react'
import { AdapterRegistry } from './components/AdapterRegistry'
import { AppIcon } from './components/AppIcon'
import { CredentialVault } from './components/CredentialVault'
import { MetricCard } from './components/MetricCard'
import { MultiProfileToolsPage } from './components/MultiProfileDock'
import { OfficialKernelCard } from './components/OfficialKernelCard'
import { PlanDrawer } from './components/PlanDrawer'
import { ProfileEditor } from './components/ProfileEditor'
import { ProfileTable } from './components/ProfileTable'
import { RecoveryWorkspace } from './components/RecoveryWorkspace'
import { RuntimePanel } from './components/RuntimePanel'
import { Sidebar, type ViewKey } from './components/Sidebar'
import { formatBytes, statusLabel } from './i18n/format'
import { ui } from './i18n'
import {
  lifecycleAllowsLaunch,
  lifecycleRecordFor,
  normalizeLifecycleBootstrap,
  type LifecycleBootstrap,
} from './lifecycle'
import { backend } from './lib/backend'
import { filterProfiles, groupsOf, profileHealth } from './lib/model'
import { isRuntimeActive, sessionForProfile } from './lib/runtime'
import type {
  AdapterImportRequest,
  AdapterInstallRequest,
  AdapterRecord,
  AdapterValidationReport,
  CredentialSaveRequest,
  KernelImportRequest,
  KernelInstallRequest,
  KernelRecord,
  LaunchPlan,
  Profile,
} from './types'

const emptyBootstrap: LifecycleBootstrap = normalizeLifecycleBootstrap({
  version: 'loading',
  profiles: [],
  providers: [],
  kernels: [],
  adapters: [],
  sessions: [],
  credentials: [],
  credentialProvider: 'Operating-system keyring',
  adapterPins: [],
  kernelPins: [],
  runtimePlatform: 'browser',
  runtimeArch: 'unknown',
})

export default function App() {
  const [data, setData] = useState<LifecycleBootstrap>(emptyBootstrap)
  const [view, setView] = useState<ViewKey>('environments')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const [group, setGroup] = useState('all')
  const [editing, setEditing] = useState<Profile>()
  const [editorOpen, setEditorOpen] = useState(false)
  const [selectedID, setSelectedID] = useState('')
  const [planProfile, setPlanProfile] = useState<Profile>()
  const [plan, setPlan] = useState<LaunchPlan>()
  const [planError, setPlanError] = useState('')
  const [runtimeBusy, setRuntimeBusy] = useState('')
  const [runtimeError, setRuntimeError] = useState('')
  const [kernelBusy, setKernelBusy] = useState(false)
  const [kernelError, setKernelError] = useState('')
  const [kernelLicenseAccepted, setKernelLicenseAccepted] = useState(false)
  const [kernelRequest, setKernelRequest] = useState<KernelImportRequest>({
    name: '',
    provider: 'custom-chromium',
    version: '148.0.0',
    sourcePath: '',
  })
  const [adapterBusy, setAdapterBusy] = useState(false)
  const [adapterError, setAdapterError] = useState('')
  const [adapterReports, setAdapterReports] = useState<Record<string, AdapterValidationReport>>({})

  async function refresh() {
    setLoading(true)
    try {
      setData(normalizeLifecycleBootstrap(await backend.bootstrap()))
      setError('')
    } catch (reason) {
      setError(errorText(reason))
    } finally {
      setLoading(false)
    }
  }

  async function refreshSessions() {
    try {
      const sessions = await backend.listSessions()
      setData((current) => ({ ...current, sessions }))
    } catch (reason) {
      setRuntimeError(errorText(reason))
    }
  }

  useEffect(() => { void refresh() }, [])
  useEffect(() => {
    if (!backend.isNative()) return
    const timer = window.setInterval(() => { void refreshSessions() }, 1500)
    return () => window.clearInterval(timer)
  }, [])

  const groups = useMemo(() => groupsOf(data.profiles), [data.profiles])
  const filtered = useMemo(() => filterProfiles(data.profiles, query, group), [data.profiles, query, group])
  const readyCount = useMemo(() => data.profiles.filter((item) => (
    profileHealth(item) === 'ready' && lifecycleAllowsLaunch(lifecycleRecordFor(data.lifecycleRecords, item.id))
  )).length, [data.profiles, data.lifecycleRecords])
  const runningCount = useMemo(() => data.sessions.filter(isRuntimeActive).length, [data.sessions])
  const attentionCount = Math.max(0, data.profiles.length - readyCount)

  const openCreate = () => {
    setEditing(undefined)
    setEditorOpen(true)
  }

  async function saveProfile(item: Profile) {
    if (item.id) await backend.updateProfile(item)
    else await backend.createProfile(item)
    await refresh()
  }

  async function cloneProfile(item: Profile) {
    await backend.cloneProfile(item.id, `${item.name} ${ui.environments.cloneSuffix}`)
    await refresh()
  }

  async function deleteProfile(item: Profile) {
    if (!window.confirm(ui.environments.moveToTrashConfirm(item.name))) return
    await backend.deleteProfile(item.id)
    await refresh()
  }

  async function saveCredential(request: CredentialSaveRequest) { await backend.saveCredential(request); await refresh() }
  async function deleteCredential(id: string) { await backend.deleteCredential(id); await refresh() }

  async function showPlan(item: Profile) {
    setPlanProfile(item)
    setPlan(undefined)
    setPlanError('')
    try { setPlan(await backend.buildLaunchPlan(item.id)) }
    catch (reason) { setPlanError(errorText(reason)) }
  }

  async function startProfile(item: Profile) {
    setRuntimeBusy(item.id)
    setRuntimeError('')
    try { await backend.startProfile(item.id) }
    catch (reason) { setRuntimeError(errorText(reason)) }
    finally { await refreshSessions(); setRuntimeBusy('') }
  }

  async function stopProfile(profileID: string) {
    setRuntimeBusy(profileID)
    setRuntimeError('')
    try { await backend.stopProfile(profileID) }
    catch (reason) { setRuntimeError(errorText(reason)) }
    finally { await refreshSessions(); setRuntimeBusy('') }
  }

  async function pickKernel() {
    setKernelError('')
    try {
      const path = await backend.pickKernelExecutable()
      if (path) setKernelRequest((current) => ({ ...current, sourcePath: path, name: current.name || basename(path) || '自定义 Chromium' }))
    } catch (reason) { setKernelError(errorText(reason)) }
  }

  async function importKernel() {
    setKernelBusy(true)
    setKernelError('')
    try {
      await backend.importKernel(kernelRequest)
      setKernelRequest((current) => ({ ...current, name: '', sourcePath: '' }))
      await refresh()
    } catch (reason) { setKernelError(errorText(reason)) }
    finally { setKernelBusy(false) }
  }

  async function installOfficialKernel(request: KernelInstallRequest) {
    setKernelBusy(true)
    setKernelError('')
    try {
      await backend.installOfficialKernel(request)
      setKernelLicenseAccepted(false)
      await refresh()
    } catch (reason) { setKernelError(errorText(reason)) }
    finally { setKernelBusy(false) }
  }

  async function verifyKernel(record: KernelRecord) {
    setKernelBusy(true)
    try { await backend.verifyKernel(record.id); await refresh() }
    catch (reason) { setKernelError(errorText(reason)) }
    finally { setKernelBusy(false) }
  }

  async function removeKernel(record: KernelRecord) {
    if (!window.confirm(ui.kernels.removeConfirm(record.name))) return
    setKernelBusy(true)
    try { await backend.deleteKernel(record.id); await refresh() }
    catch (reason) { setKernelError(errorText(reason)) }
    finally { setKernelBusy(false) }
  }

  async function pickAdapter() {
    try { return await backend.pickAdapterExecutable() }
    catch (reason) { setAdapterError(errorText(reason)); return '' }
  }

  async function importAdapter(request: AdapterImportRequest) {
    setAdapterBusy(true)
    setAdapterError('')
    try { await backend.importAdapter(request); await refresh() }
    catch (reason) { setAdapterError(errorText(reason)) }
    finally { setAdapterBusy(false) }
  }

  async function verifyAdapter(record: AdapterRecord) {
    setAdapterBusy(true)
    try { await backend.verifyAdapter(record.id); await refresh() }
    catch (reason) { setAdapterError(errorText(reason)) }
    finally { setAdapterBusy(false) }
  }

  async function validateAdapter(record: AdapterRecord) {
    setAdapterBusy(true)
    setAdapterError('')
    try {
      const report = await backend.validateAdapter(record.id)
      setAdapterReports((current) => ({ ...current, [record.id]: report }))
      await refresh()
    } catch (reason) { setAdapterError(errorText(reason)) }
    finally { setAdapterBusy(false) }
  }

  async function installOfficialAdapter(request: AdapterInstallRequest) {
    setAdapterBusy(true)
    setAdapterError('')
    try { await backend.installOfficialAdapter(request); await refresh() }
    catch (reason) { setAdapterError(errorText(reason)) }
    finally { setAdapterBusy(false) }
  }

  async function removeAdapter(record: AdapterRecord) {
    if (!window.confirm(`确定移除代理组件“${record.name}”吗？`)) return
    setAdapterBusy(true)
    try { await backend.deleteAdapter(record.id); await refresh() }
    catch (reason) { setAdapterError(errorText(reason)) }
    finally { setAdapterBusy(false) }
  }

  const table = (profiles: Profile[]) => <ProfileTable
    profiles={profiles}
    sessions={data.sessions}
    lifecycleRecords={data.lifecycleRecords}
    selectedID={selectedID}
    nativeMode={backend.isNative()}
    busyProfileID={runtimeBusy}
    emptyKind={data.profiles.length === 0 ? 'first-use' : 'search'}
    onCreate={openCreate}
    onSelect={(item) => setSelectedID(item.id)}
    onEdit={(item) => { setEditing(item); setEditorOpen(true) }}
    onClone={(item) => void cloneProfile(item)}
    onPlan={(item) => void showPlan(item)}
    onStart={(item) => void startProfile(item)}
    onStop={(item) => void stopProfile(item.id)}
    onDelete={(item) => void deleteProfile(item)}
  />

  function environments() {
    return <>
      <Heading eyebrow={ui.environments.eyebrow} title={ui.environments.title} description={ui.environments.description} action={<button className="button primary" onClick={openCreate}><AppIcon name="add" />{ui.common.create}</button>} />
      <div className="metric-grid environment-metrics">
        <MetricCard label={ui.environments.total} value={data.profiles.length} detail={ui.environments.totalDetail} />
        <MetricCard label={ui.environments.ready} value={readyCount} detail={ui.environments.readyDetail} tone="good" />
        <MetricCard label={ui.environments.running} value={runningCount} detail={ui.environments.runningDetail} tone={runningCount ? 'good' : 'neutral'} />
        <MetricCard label={ui.environments.attention} value={attentionCount} detail={ui.environments.attentionDetail} tone={attentionCount ? 'warn' : 'neutral'} />
      </div>
      {runtimeError && <div className="form-error page-form-error">{runtimeError}</div>}
      <section className="panel environment-workspace">
        <div className="toolbar environment-toolbar">
          <div className="search-box"><AppIcon name="search" size={16} /><input value={query} onChange={(event) => setQuery(event.target.value)} placeholder={ui.environments.searchPlaceholder} /></div>
          <select value={group} onChange={(event) => setGroup(event.target.value)}><option value="all">{ui.environments.allGroups}</option>{groups.map((item) => <option key={item}>{item}</option>)}</select>
          <span className="result-count">{ui.environments.resultCount(filtered.length)}</span>
        </div>
        {table(filtered)}
      </section>
    </>
  }

  function network() {
    return <>
      <Heading eyebrow={ui.network.eyebrow} title={ui.network.title} description={ui.network.description} />
      <div className="metric-grid environment-metrics network-metrics">
        <MetricCard label={ui.network.adapters} value={data.adapters.length} detail="已注册的 Xray 与 sing-box 组件" />
        <MetricCard label={ui.network.credentials} value={data.credentials.length} detail="操作系统凭据存储中的引用" />
        <MetricCard label={ui.network.sessions} value={runningCount} detail="当前浏览器与代理运行会话" tone={runningCount ? 'good' : 'neutral'} />
        <MetricCard label="自动下载" value="关闭" detail="敏感运行组件不会静默更新" />
      </div>
      <AdapterRegistry
        records={data.adapters}
        pins={data.adapterPins}
        reports={adapterReports}
        runtimePlatform={data.runtimePlatform}
        runtimeArch={data.runtimeArch}
        nativeMode={backend.isNative()}
        busy={adapterBusy}
        error={adapterError}
        onPick={pickAdapter}
        onImport={importAdapter}
        onInstall={installOfficialAdapter}
        onVerify={verifyAdapter}
        onValidate={validateAdapter}
        onDelete={removeAdapter}
      />
    </>
  }

  function runtime() {
    return <>
      <Heading eyebrow={ui.runtime.eyebrow} title={ui.runtime.title} description={ui.runtime.description} action={<button className="button secondary" onClick={() => void refreshSessions()}><AppIcon name="refresh" />{ui.runtime.refresh}</button>} />
      {!backend.isNative() && <div className="info-banner runtime-mode-note"><strong>{ui.runtime.desktopRequired}</strong><p>{ui.runtime.previewNote}</p></div>}
      {runtimeError && <div className="form-error page-form-error">{runtimeError}</div>}
      <RuntimePanel sessions={data.sessions} nativeMode={backend.isNative()} busyProfileID={runtimeBusy} onStop={(profileID) => void stopProfile(profileID)} />
    </>
  }

  function kernels() {
    const pin = data.kernelPins[0]
    const importProviders = data.providers.filter((item) => item.id !== pin?.providerId)
    const provider = importProviders.find((item) => item.id === kernelRequest.provider)
    return <>
      <Heading eyebrow={ui.kernels.eyebrow} title={ui.kernels.title} description={ui.kernels.description} />
      <OfficialKernelCard
        pin={pin}
        records={data.kernels}
        runtimePlatform={data.runtimePlatform}
        runtimeArch={data.runtimeArch}
        nativeMode={backend.isNative()}
        busy={kernelBusy}
        accepted={kernelLicenseAccepted}
        onAcceptedChange={setKernelLicenseAccepted}
        onInstall={(request) => void installOfficialKernel(request)}
      />
      <section className="panel kernel-import">
        <div className="panel-heading"><div><h2>{ui.kernels.customTitle}</h2><p>{ui.kernels.customDescription}</p></div></div>
        <div className="kernel-import-grid">
          <label>{ui.kernels.name}<input value={kernelRequest.name} onChange={(event) => setKernelRequest((current) => ({ ...current, name: event.target.value }))} /></label>
          <label>{ui.kernels.provider}<select value={kernelRequest.provider} onChange={(event) => { const next = importProviders.find((item) => item.id === event.target.value); setKernelRequest((current) => ({ ...current, provider: event.target.value, version: next?.versions[0] || '' })) }}>{importProviders.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <label>{ui.kernels.version}<select value={kernelRequest.version} onChange={(event) => setKernelRequest((current) => ({ ...current, version: event.target.value }))}>{provider?.versions.map((version) => <option key={version}>{version}</option>)}</select></label>
          <label className="kernel-path">{ui.kernels.executablePath}<div className="path-picker"><input readOnly value={kernelRequest.sourcePath} /><button type="button" className="button secondary" onClick={() => void pickKernel()} disabled={!backend.isNative()}>{ui.common.browse}</button></div></label>
          <button className="button primary kernel-import-button" onClick={() => void importKernel()} disabled={kernelBusy || !backend.isNative() || !kernelRequest.name.trim() || !kernelRequest.sourcePath.trim()}>{kernelBusy ? ui.common.working : ui.kernels.importAndHash}</button>
        </div>
        {kernelError && <div className="form-error">{kernelError}</div>}
      </section>
      <div className="kernel-registry-list">
        {data.kernels.length === 0 ? <Empty title={ui.kernels.emptyTitle} detail={ui.kernels.emptyDetail} /> : data.kernels.map((record) => <article className="kernel-record" key={record.id}>
          <div className="kernel-record-head"><div className="kernel-symbol"><AppIcon name="kernel" /></div><div><h2>{record.name}</h2><code>{record.provider} · Chromium {record.version}</code></div><span className={`kernel-status ${record.status}`}>{statusLabel(record.status)}</span></div>
          <dl><div><dt>{ui.kernels.executableSha}</dt><dd>{record.sha256.slice(0, 16)}…{record.sha256.slice(-8)}</dd></div>{record.packageTreeSha256 && <div><dt>{ui.kernels.packageTree}</dt><dd>{record.packageFileCount} 个文件 · {record.packageTreeSha256.slice(0, 16)}…</dd></div>}<div><dt>{ui.kernels.size}</dt><dd>{formatBytes(record.sizeBytes)}</dd></div><div><dt>{ui.kernels.managedPath}</dt><dd title={record.executable}>{record.executable}</dd></div></dl>
          <div className="kernel-actions"><button className="button secondary" onClick={() => void verifyKernel(record)} disabled={kernelBusy}>{ui.common.verify}</button><button className="button secondary danger-text" onClick={() => void removeKernel(record)} disabled={kernelBusy}>{ui.common.remove}</button></div>
        </article>)}
      </div>
    </>
  }

  function credentials() {
    return <>
      <Heading eyebrow={ui.credentials.eyebrow} title={ui.credentials.title} description={ui.credentials.description} />
      <CredentialVault records={data.credentials} provider={data.credentialProvider} nativeMode={backend.isNative()} onSave={saveCredential} onDelete={deleteCredential} />
    </>
  }

  function settings() {
    return <>
      <Heading eyebrow={ui.settings.eyebrow} title={ui.settings.title} description={ui.settings.description} />
      <section className="settings-grid product-settings-grid">
        <article className="panel setting-card"><h2>{ui.settings.runtime}</h2><dl><div><dt>{ui.settings.version}</dt><dd>{data.version}</dd></div><div><dt>{ui.settings.frontendMode}</dt><dd>{backend.isNative() ? 'Wails 桌面应用' : ui.app.browserPreview}</dd></div><div><dt>{ui.settings.activeSessions}</dt><dd>{runningCount}</dd></div></dl><button className="button secondary settings-action" onClick={() => setView('runtime')}>{ui.settings.openRuntime}</button></article>
        <article className="panel setting-card"><h2>{ui.settings.dependencies}</h2><dl><div><dt>{ui.settings.kernels}</dt><dd>{data.kernels.length}</dd></div><div><dt>{ui.settings.adapters}</dt><dd>{data.adapters.length}</dd></div><div><dt>{ui.settings.automaticDownloads}</dt><dd>{ui.settings.disabled}</dd></div></dl><button className="button secondary settings-action" onClick={() => setView('kernels')}>{ui.settings.openKernels}</button></article>
        <article className="panel setting-card"><h2>{ui.settings.safety}</h2><ul className="plain-list"><li>同机快照和恢复已启用，并保持验证与回滚边界。</li><li>归档和可恢复回收站使用权威生命周期操作。</li><li>密码只保存在操作系统凭据存储中。</li><li>便携定义、模板和批量操作不复制秘密或浏览器数据。</li></ul><button className="button secondary settings-action" onClick={() => setView('credentials')}>{ui.settings.openCredentials}</button></article>
      </section>
    </>
  }

  const activeEditingSession = editing ? sessionForProfile(data.sessions, editing.id) : undefined
  return <div className="app-shell">
    <Sidebar active={view} onChange={setView} nativeMode={backend.isNative()} />
    <main className="main-content">
      <div className="topbar">
        <div className="window-context"><span className="context-dot" /><strong>{pageTitle(view)}</strong></div>
        <div className="top-actions"><span className="version-chip">v{data.version}</span><button title={ui.app.refresh} onClick={() => void refresh()} aria-label={ui.app.refresh}><AppIcon name="refresh" size={17} /></button></div>
      </div>
      <div className="page-content">
        {loading && <div className="loading-screen">{ui.app.loading}</div>}
        {error && <div className="form-error page-form-error">{error}</div>}
        {!loading && view === 'environments' && environments()}
        {!loading && view === 'network' && network()}
        {!loading && view === 'recovery' && <RecoveryWorkspace data={data} onRefresh={refresh} />}
        {!loading && view === 'batch' && <MultiProfileToolsPage />}
        {!loading && view === 'settings' && settings()}
        {!loading && view === 'runtime' && runtime()}
        {!loading && view === 'kernels' && kernels()}
        {!loading && view === 'credentials' && credentials()}
      </div>
    </main>
    <ProfileEditor open={editorOpen && !isRuntimeActive(activeEditingSession)} profile={editing} providers={data.providers} kernels={data.kernels} adapters={data.adapters} credentials={data.credentials} onClose={() => setEditorOpen(false)} onSave={saveProfile} />
    <PlanDrawer profile={planProfile} plan={plan} error={planError} onClose={() => { setPlanProfile(undefined); setPlan(undefined) }} />
  </div>
}

function Heading({ eyebrow, title, description, action }: { eyebrow: string; title: string; description: string; action?: React.ReactNode }) {
  return <div className="page-heading compact"><div><span className="eyebrow">{eyebrow}</span><h1>{title}</h1><p>{description}</p></div>{action}</div>
}

function Empty({ title, detail }: { title: string; detail: string }) {
  return <section className="panel empty-state"><div className="empty-icon"><AppIcon name="kernel" size={25} /></div><h3>{title}</h3><p>{detail}</p></section>
}

function pageTitle(view: ViewKey): string {
  const titles: Record<ViewKey, string> = {
    environments: ui.nav.environments,
    network: ui.nav.network,
    recovery: ui.nav.recovery,
    batch: ui.nav.batch,
    settings: ui.nav.settings,
    runtime: ui.nav.runtime,
    kernels: ui.nav.kernels,
    credentials: ui.nav.credentials,
  }
  return titles[view]
}

function errorText(reason: unknown): string {
  return reason instanceof Error ? reason.message : String(reason)
}

function basename(path: string): string {
  return path.split(/[\\/]/).pop() || ''
}
