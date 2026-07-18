import { useEffect, useMemo, useState } from 'react'
import { AdapterRegistry } from './components/AdapterRegistry'
import { CredentialVault } from './components/CredentialVault'
import { MetricCard } from './components/MetricCard'
import { PlanDrawer } from './components/PlanDrawer'
import { ProfileEditor } from './components/ProfileEditor'
import { ProfileTable } from './components/ProfileTable'
import { RuntimePanel } from './components/RuntimePanel'
import { Sidebar, type ViewKey } from './components/Sidebar'
import { backend } from './lib/backend'
import { filterProfiles, groupsOf, profileHealth } from './lib/model'
import { isRuntimeActive, sessionForProfile } from './lib/runtime'
import type {
  AdapterImportRequest,
  AdapterInstallRequest,
  AdapterRecord,
  AdapterValidationReport,
  Bootstrap,
  CredentialSaveRequest,
  KernelImportRequest,
  KernelRecord,
  LaunchPlan,
  Profile,
} from './types'

const emptyBootstrap: Bootstrap = {
  version: 'loading',
  profiles: [],
  providers: [],
  kernels: [],
  adapters: [],
  sessions: [],
  credentials: [],
  credentialProvider: 'Operating-system keyring',
  adapterPins: [],
  runtimePlatform: 'browser',
  runtimeArch: 'unknown',
}

export default function App() {
  const [data, setData] = useState<Bootstrap>(emptyBootstrap)
  const [view, setView] = useState<ViewKey>('dashboard')
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
  const [kernelRequest, setKernelRequest] = useState<KernelImportRequest>({
    name: '',
    provider: 'patched-chromium',
    version: '148.0.0',
    sourcePath: '',
  })
  const [adapterBusy, setAdapterBusy] = useState(false)
  const [adapterError, setAdapterError] = useState('')
  const [adapterReports, setAdapterReports] = useState<Record<string, AdapterValidationReport>>({})

  async function refresh() {
    setLoading(true)
    try {
      setData(await backend.bootstrap())
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
  const readyCount = useMemo(() => data.profiles.filter((item) => profileHealth(item) === 'ready').length, [data.profiles])
  const runningCount = useMemo(() => data.sessions.filter(isRuntimeActive).length, [data.sessions])

  async function saveProfile(item: Profile) { if (item.id) await backend.updateProfile(item); else await backend.createProfile(item); await refresh() }
  async function cloneProfile(item: Profile) { await backend.cloneProfile(item.id, `${item.name} Copy`); await refresh() }
  async function deleteProfile(item: Profile) { if (!window.confirm(`Delete “${item.name}”? Browser data is not removed.`)) return; await backend.deleteProfile(item.id); await refresh() }
  async function saveCredential(request: CredentialSaveRequest) { await backend.saveCredential(request); await refresh() }
  async function deleteCredential(id: string) { await backend.deleteCredential(id); await refresh() }

  async function showPlan(item: Profile) {
    setPlanProfile(item); setPlan(undefined); setPlanError('')
    try { setPlan(await backend.buildLaunchPlan(item.id)) }
    catch (reason) { setPlanError(errorText(reason)) }
  }

  async function startProfile(item: Profile) {
    setRuntimeBusy(item.id); setRuntimeError('')
    try { await backend.startProfile(item.id) }
    catch (reason) { setRuntimeError(errorText(reason)) }
    finally { await refreshSessions(); setRuntimeBusy('') }
  }

  async function stopProfile(profileID: string) {
    setRuntimeBusy(profileID); setRuntimeError('')
    try { await backend.stopProfile(profileID) }
    catch (reason) { setRuntimeError(errorText(reason)) }
    finally { await refreshSessions(); setRuntimeBusy('') }
  }

  async function pickKernel() {
    setKernelError('')
    try {
      const path = await backend.pickKernelExecutable()
      if (path) setKernelRequest((current) => ({ ...current, sourcePath: path, name: current.name || basename(path) || 'Chromium kernel' }))
    } catch (reason) { setKernelError(errorText(reason)) }
  }

  async function importKernel() {
    setKernelBusy(true); setKernelError('')
    try { await backend.importKernel(kernelRequest); setKernelRequest((current) => ({ ...current, name: '', sourcePath: '' })); await refresh() }
    catch (reason) { setKernelError(errorText(reason)) }
    finally { setKernelBusy(false) }
  }
  async function verifyKernel(record: KernelRecord) { setKernelBusy(true); try { await backend.verifyKernel(record.id); await refresh() } catch (reason) { setKernelError(errorText(reason)) } finally { setKernelBusy(false) } }
  async function removeKernel(record: KernelRecord) { if (!window.confirm(`Remove “${record.name}”?`)) return; setKernelBusy(true); try { await backend.deleteKernel(record.id); await refresh() } catch (reason) { setKernelError(errorText(reason)) } finally { setKernelBusy(false) } }

  async function pickAdapter() { try { return await backend.pickAdapterExecutable() } catch (reason) { setAdapterError(errorText(reason)); return '' } }
  async function importAdapter(request: AdapterImportRequest) { setAdapterBusy(true); setAdapterError(''); try { await backend.importAdapter(request); await refresh() } catch (reason) { setAdapterError(errorText(reason)) } finally { setAdapterBusy(false) } }
  async function verifyAdapter(record: AdapterRecord) { setAdapterBusy(true); try { await backend.verifyAdapter(record.id); await refresh() } catch (reason) { setAdapterError(errorText(reason)) } finally { setAdapterBusy(false) } }
  async function validateAdapter(record: AdapterRecord) { setAdapterBusy(true); setAdapterError(''); try { const report = await backend.validateAdapter(record.id); setAdapterReports((current) => ({ ...current, [record.id]: report })); await refresh() } catch (reason) { setAdapterError(errorText(reason)) } finally { setAdapterBusy(false) } }
  async function installOfficialAdapter(request: AdapterInstallRequest) { setAdapterBusy(true); setAdapterError(''); try { await backend.installOfficialAdapter(request); await refresh() } catch (reason) { setAdapterError(errorText(reason)) } finally { setAdapterBusy(false) } }

  async function removeAdapter(record: AdapterRecord) { if (!window.confirm(`Remove “${record.name}”?`)) return; setAdapterBusy(true); try { await backend.deleteAdapter(record.id); await refresh() } catch (reason) { setAdapterError(errorText(reason)) } finally { setAdapterBusy(false) } }

  const table = (profiles: Profile[]) => <ProfileTable
    profiles={profiles}
    sessions={data.sessions}
    selectedID={selectedID}
    nativeMode={backend.isNative()}
    busyProfileID={runtimeBusy}
    onSelect={(item) => setSelectedID(item.id)}
    onEdit={(item) => { setEditing(item); setEditorOpen(true) }}
    onClone={(item) => void cloneProfile(item)}
    onPlan={(item) => void showPlan(item)}
    onStart={(item) => void startProfile(item)}
    onStop={(item) => void stopProfile(item.id)}
    onDelete={(item) => void deleteProfile(item)}
  />

  function dashboard() {
    return <>
      <Heading eyebrow="Local identity workspace" title="Browser environments, without the guesswork." description="Every profile uses explicit kernel, identity, network and managed dependency contracts." action={<button className="button primary" onClick={() => { setEditing(undefined); setEditorOpen(true) }}>＋ New profile</button>} />
      <div className="metric-grid">
        <MetricCard label="Profiles" value={data.profiles.length} detail="Isolated local identities" />
        <MetricCard label="Ready" value={readyCount} detail="Passed visible checks" tone="good" />
        <MetricCard label="Running" value={runningCount} detail="Supervised sessions" tone={runningCount ? 'good' : 'neutral'} />
        <MetricCard label="Adapters" value={data.adapters.length} detail="Managed Xray and sing-box binaries" />
      </div>
      {runtimeError && <div className="form-error runtime-global-error">{runtimeError}</div>}
      <div className="dashboard-grid">
        <section className="panel wide"><div className="panel-heading"><div><h2>Recent profiles</h2><p>Start only after referenced binaries pass integrity checks.</p></div><button className="text-button" onClick={() => setView('profiles')}>View all →</button></div>{table(data.profiles.slice(0, 5))}</section>
        <section className="panel rail-card"><div className="panel-heading"><div><h2>Safety posture</h2><p>Runtime boundaries that cannot silently weaken.</p></div></div><ul className="check-list"><li><span>✓</span><div><strong>Verified browser kernels</strong><p>Legacy executable paths stay dry-run only.</p></div></li><li><span>✓</span><div><strong>OS-backed credentials</strong><p>Passwords never enter profile metadata.</p></div></li><li><span>✓</span><div><strong>Authenticated loopback bridge</strong><p>HTTP, HTTPS and SOCKS5 secrets stay out of Chromium arguments.</p></div></li><li><span>✓</span><div><strong>Supervised advanced runtimes</strong><p>Xray and sing-box routes use private per-session configuration.</p></div></li></ul></section>
      </div>
    </>
  }

  function profiles() {
    return <><Heading eyebrow="Identity registry" title="Browser profiles" description="Start, stop, diagnose, clone and edit isolated environments." action={<button className="button primary" onClick={() => { setEditing(undefined); setEditorOpen(true) }}>＋ New profile</button>} />{runtimeError && <div className="form-error runtime-global-error">{runtimeError}</div>}<section className="panel"><div className="toolbar"><div className="search-box">⌕<input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search name, tag, kernel or proxy…" /></div><select value={group} onChange={(event) => setGroup(event.target.value)}><option value="all">All groups</option>{groups.map((item) => <option key={item}>{item}</option>)}</select><span className="result-count">{filtered.length} profile{filtered.length === 1 ? '' : 's'}</span></div>{table(filtered)}</section></>
  }

  function runtime() {
    return <><Heading eyebrow="Local process supervisor" title="Runtime sessions" description="Process state, loopback CDP readiness, logs and exits remain local." action={<button className="button secondary" onClick={() => void refreshSessions()}>Refresh sessions</button>} />{!backend.isNative() && <div className="info-banner runtime-mode-note"><strong>Desktop runtime required</strong><p>Browser preview mode cannot start local processes.</p></div>}{runtimeError && <div className="form-error runtime-global-error">{runtimeError}</div>}<RuntimePanel sessions={data.sessions} nativeMode={backend.isNative()} busyProfileID={runtimeBusy} onStop={(profileID) => void stopProfile(profileID)} /></>
  }

  function kernels() {
    const provider = data.providers.find((item) => item.id === kernelRequest.provider)
    return <><Heading eyebrow="Verified local binaries" title="Kernel registry" description="Import Chromium into private managed storage with an explicit provider contract." /><section className="panel kernel-import"><div className="panel-heading"><div><h2>Register local kernel</h2><p>Symbolic links, directories and empty files are rejected.</p></div></div><div className="kernel-import-grid"><label>Name<input value={kernelRequest.name} onChange={(event) => setKernelRequest((current) => ({ ...current, name: event.target.value }))} /></label><label>Provider<select value={kernelRequest.provider} onChange={(event) => { const next = data.providers.find((item) => item.id === event.target.value); setKernelRequest((current) => ({ ...current, provider: event.target.value, version: next?.versions[0] || '' })) }}>{data.providers.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label><label>Version<select value={kernelRequest.version} onChange={(event) => setKernelRequest((current) => ({ ...current, version: event.target.value }))}>{provider?.versions.map((version) => <option key={version}>{version}</option>)}</select></label><label className="kernel-path">Executable path<div className="path-picker"><input readOnly value={kernelRequest.sourcePath} /><button type="button" className="button secondary" onClick={() => void pickKernel()} disabled={!backend.isNative()}>Browse</button></div></label><button className="button primary kernel-import-button" onClick={() => void importKernel()} disabled={kernelBusy || !backend.isNative() || !kernelRequest.name.trim() || !kernelRequest.sourcePath.trim()}>{kernelBusy ? 'Working…' : 'Import and hash'}</button></div>{kernelError && <div className="form-error">{kernelError}</div>}</section><div className="kernel-registry-list">{data.kernels.length === 0 ? <Empty icon="⬡" title="No registered kernels" detail="Import a Chromium executable to create the first managed record." /> : data.kernels.map((record) => <article className="kernel-record" key={record.id}><div className="kernel-record-head"><div className="kernel-symbol">⬡</div><div><h2>{record.name}</h2><code>{record.provider} · Chromium {record.version.split('.')[0]}</code></div><span className={`kernel-status ${record.status}`}>{record.status}</span></div><dl><div><dt>SHA-256</dt><dd>{record.sha256.slice(0, 16)}…{record.sha256.slice(-8)}</dd></div><div><dt>Size</dt><dd>{(record.sizeBytes / 1024 / 1024).toFixed(1)} MB</dd></div><div><dt>Managed path</dt><dd title={record.executable}>{record.executable}</dd></div></dl><div className="kernel-actions"><button className="button secondary" onClick={() => void verifyKernel(record)} disabled={kernelBusy}>Verify</button><button className="button secondary danger-text" onClick={() => void removeKernel(record)} disabled={kernelBusy}>Remove</button></div></article>)}</div></>
  }

  const adapters = () => <><Heading eyebrow="Managed external runtimes" title="Proxy adapters" description="Register local Xray or sing-box binaries, identify exact pinned official releases, and run their native configuration checks before production use." /><AdapterRegistry records={data.adapters} pins={data.adapterPins} reports={adapterReports} runtimePlatform={data.runtimePlatform} runtimeArch={data.runtimeArch} nativeMode={backend.isNative()} busy={adapterBusy} error={adapterError} onPick={pickAdapter} onImport={importAdapter} onInstall={installOfficialAdapter} onVerify={verifyAdapter} onValidate={validateAdapter} onDelete={removeAdapter} /></>
  const credentials = () => <><Heading eyebrow="Operating-system secret storage" title="Credential vault" description={`Passwords stay inside ${data.credentialProvider}.`} /><CredentialVault records={data.credentials} provider={data.credentialProvider} nativeMode={backend.isNative()} onSave={saveCredential} onDelete={deleteCredential} /></>
  const settings = () => <><Heading eyebrow="Application controls" title="Settings" description="Xray and sing-box execution are enabled for reviewed protocol subsets; unsupported options remain gated." /><section className="settings-grid"><article className="panel setting-card"><h2>Runtime</h2><dl><div><dt>Application version</dt><dd>{data.version}</dd></div><div><dt>Frontend mode</dt><dd>{backend.isNative() ? 'Wails desktop' : 'Browser preview'}</dd></div><div><dt>Active sessions</dt><dd>{runningCount}</dd></div></dl></article><article className="panel setting-card"><h2>Managed dependencies</h2><dl><div><dt>Kernels</dt><dd>{data.kernels.length}</dd></div><div><dt>Proxy adapters</dt><dd>{data.adapters.length}</dd></div><div><dt>Automatic downloads</dt><dd>Disabled</dd></div></dl></article><article className="panel setting-card"><h2>Deferred work</h2><ul className="plain-list"><li>Broader sing-box and Xray share-link compatibility</li><li>Additional reviewed transports and protocol options</li><li>Optional signed installer manifests</li><li>Encrypted export/import</li></ul></article></section></>

  const activeEditingSession = editing ? sessionForProfile(data.sessions, editing.id) : undefined
  return <div className="app-shell"><Sidebar active={view} onChange={setView} nativeMode={backend.isNative()} /><main className="main-content"><div className="topbar"><div className="window-context"><span className="context-dot" />Veilium workspace</div><div className="top-actions"><span className="version-chip">v{data.version}</span><button title="Refresh" onClick={() => void refresh()}>↻</button></div></div><div className="page-content">{loading && <div className="loading-screen">Loading isolated identities…</div>}{error && <div className="form-error">{error}</div>}{!loading && view === 'dashboard' && dashboard()}{!loading && view === 'profiles' && profiles()}{!loading && view === 'runtime' && runtime()}{!loading && view === 'kernels' && kernels()}{!loading && view === 'adapters' && adapters()}{!loading && view === 'credentials' && credentials()}{!loading && view === 'settings' && settings()}</div></main><ProfileEditor open={editorOpen && !isRuntimeActive(activeEditingSession)} profile={editing} providers={data.providers} kernels={data.kernels} adapters={data.adapters} credentials={data.credentials} onClose={() => setEditorOpen(false)} onSave={saveProfile} /><PlanDrawer profile={planProfile} plan={plan} error={planError} onClose={() => { setPlanProfile(undefined); setPlan(undefined) }} /></div>
}

function Heading({ eyebrow, title, description, action }: { eyebrow: string; title: string; description: string; action?: React.ReactNode }) {
  return <div className="page-heading compact"><div><span className="eyebrow">{eyebrow}</span><h1>{title}</h1><p>{description}</p></div>{action}</div>
}
function Empty({ icon, title, detail }: { icon: string; title: string; detail: string }) {
  return <section className="panel empty-state"><div className="empty-icon">{icon}</div><h3>{title}</h3><p>{detail}</p></section>
}
function errorText(reason: unknown): string { return reason instanceof Error ? reason.message : String(reason) }
function basename(path: string): string { return path.split(/[\\/]/).pop() || '' }
