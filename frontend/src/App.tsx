import { useEffect, useMemo, useState } from 'react'
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
import type { Bootstrap, CredentialSaveRequest, KernelImportRequest, KernelRecord, LaunchPlan, Profile } from './types'

const emptyBootstrap: Bootstrap = {
  version: 'loading',
  profiles: [],
  providers: [],
  kernels: [],
  sessions: [],
  credentials: [],
  credentialProvider: 'Operating-system keyring',
}

export default function App() {
  const [bootstrap, setBootstrap] = useState<Bootstrap>(emptyBootstrap)
  const [view, setView] = useState<ViewKey>('dashboard')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [query, setQuery] = useState('')
  const [group, setGroup] = useState('all')
  const [editorOpen, setEditorOpen] = useState(false)
  const [editing, setEditing] = useState<Profile | undefined>()
  const [selected, setSelected] = useState<Profile | undefined>()
  const [planProfile, setPlanProfile] = useState<Profile | undefined>()
  const [plan, setPlan] = useState<LaunchPlan | undefined>()
  const [planError, setPlanError] = useState('')
  const [kernelRequest, setKernelRequest] = useState<KernelImportRequest>({ name: '', provider: 'patched-chromium', version: '148.0.0', sourcePath: '' })
  const [kernelBusy, setKernelBusy] = useState(false)
  const [kernelError, setKernelError] = useState('')
  const [runtimeBusyProfileID, setRuntimeBusyProfileID] = useState('')
  const [runtimeError, setRuntimeError] = useState('')

  async function refresh() {
    setLoading(true)
    try {
      setBootstrap(await backend.bootstrap())
      setError('')
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setLoading(false)
    }
  }

  async function refreshSessions() {
    try {
      const sessions = await backend.listSessions()
      setBootstrap((current) => ({ ...current, sessions }))
    } catch (reason) {
      setRuntimeError(reason instanceof Error ? reason.message : String(reason))
    }
  }

  useEffect(() => { void refresh() }, [])
  useEffect(() => {
    if (!backend.isNative()) return
    const timer = window.setInterval(() => { void refreshSessions() }, 1500)
    return () => window.clearInterval(timer)
  }, [])

  const filtered = useMemo(() => filterProfiles(bootstrap.profiles, query, group), [bootstrap.profiles, query, group])
  const groups = useMemo(() => groupsOf(bootstrap.profiles), [bootstrap.profiles])
  const metrics = useMemo(() => {
    const ready = bootstrap.profiles.filter((item) => profileHealth(item) === 'ready').length
    const running = bootstrap.sessions.filter((item) => isRuntimeActive(item)).length
    return { total: bootstrap.profiles.length, ready, running, warnings: bootstrap.profiles.length - ready }
  }, [bootstrap.profiles, bootstrap.sessions])

  async function saveProfile(item: Profile) {
    if (item.id) await backend.updateProfile(item)
    else await backend.createProfile(item)
    await refresh()
  }

  async function cloneProfile(item: Profile) {
    await backend.cloneProfile(item.id, `${item.name} Copy`)
    await refresh()
  }

  async function deleteProfile(item: Profile) {
    if (!window.confirm(`Delete “${item.name}”? Browser data is not removed.`)) return
    await backend.deleteProfile(item.id)
    if (selected?.id === item.id) setSelected(undefined)
    await refresh()
  }

  async function saveCredential(request: CredentialSaveRequest) {
    await backend.saveCredential(request)
    await refresh()
  }

  async function deleteCredential(id: string) {
    await backend.deleteCredential(id)
    await refresh()
  }

  async function showPlan(item: Profile) {
    setPlanProfile(item)
    setPlan(undefined)
    setPlanError('')
    try {
      setPlan(await backend.buildLaunchPlan(item.id))
    } catch (reason) {
      setPlanError(reason instanceof Error ? reason.message : String(reason))
    }
  }

  async function startProfile(item: Profile) {
    setRuntimeBusyProfileID(item.id)
    setRuntimeError('')
    try {
      await backend.startProfile(item.id)
      await refreshSessions()
    } catch (reason) {
      setRuntimeError(reason instanceof Error ? reason.message : String(reason))
      await refreshSessions()
    } finally {
      setRuntimeBusyProfileID('')
    }
  }

  async function stopProfileByID(profileID: string) {
    setRuntimeBusyProfileID(profileID)
    setRuntimeError('')
    try {
      await backend.stopProfile(profileID)
      await refreshSessions()
    } catch (reason) {
      setRuntimeError(reason instanceof Error ? reason.message : String(reason))
      await refreshSessions()
    } finally {
      setRuntimeBusyProfileID('')
    }
  }

  function openCreate() {
    setEditing(undefined)
    setEditorOpen(true)
  }

  async function chooseKernel() {
    setKernelError('')
    try {
      const path = await backend.pickKernelExecutable()
      if (path) setKernelRequest((current) => ({ ...current, sourcePath: path, name: current.name || path.split(/[\\/]/).pop() || 'Chromium kernel' }))
    } catch (reason) {
      setKernelError(reason instanceof Error ? reason.message : String(reason))
    }
  }

  async function importKernel() {
    setKernelBusy(true)
    setKernelError('')
    try {
      await backend.importKernel(kernelRequest)
      setKernelRequest((current) => ({ ...current, name: '', sourcePath: '' }))
      await refresh()
    } catch (reason) {
      setKernelError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setKernelBusy(false)
    }
  }

  async function verifyKernel(record: KernelRecord) {
    setKernelBusy(true)
    setKernelError('')
    try {
      await backend.verifyKernel(record.id)
      await refresh()
    } catch (reason) {
      setKernelError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setKernelBusy(false)
    }
  }

  async function deleteKernel(record: KernelRecord) {
    if (!window.confirm(`Remove “${record.name}” from managed storage?`)) return
    setKernelBusy(true)
    setKernelError('')
    try {
      await backend.deleteKernel(record.id)
      await refresh()
    } catch (reason) {
      setKernelError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setKernelBusy(false)
    }
  }

  const profileTable = (profiles: Profile[]) => (
    <ProfileTable
      profiles={profiles}
      sessions={bootstrap.sessions}
      selectedID={selected?.id}
      nativeMode={backend.isNative()}
      busyProfileID={runtimeBusyProfileID}
      onSelect={setSelected}
      onEdit={(item) => { setEditing(item); setEditorOpen(true) }}
      onClone={(item) => void cloneProfile(item)}
      onPlan={(item) => void showPlan(item)}
      onStart={(item) => void startProfile(item)}
      onStop={(item) => void stopProfileByID(item.id)}
      onDelete={(item) => void deleteProfile(item)}
    />
  )

  function renderDashboard() {
    return <>
      <div className="page-heading">
        <div><span className="eyebrow">Local identity workspace</span><h1>Browser environments, without the guesswork.</h1><p>Every profile uses an explicit kernel contract, isolated data directory, reviewable network route and supervised local process.</p></div>
        <button className="button primary" onClick={openCreate}>＋ New profile</button>
      </div>
      <div className="metric-grid">
        <MetricCard label="Profiles" value={metrics.total} detail="Isolated local identities" />
        <MetricCard label="Ready" value={metrics.ready} detail="Passed visible configuration checks" tone="good" />
        <MetricCard label="Running" value={metrics.running} detail="Supervised local browser sessions" tone={metrics.running ? 'good' : 'neutral'} />
        <MetricCard label="Vault items" value={bootstrap.credentials.length} detail={`References backed by ${bootstrap.credentialProvider}`} tone="good" />
      </div>
      {runtimeError && <div className="form-error runtime-global-error">{runtimeError}</div>}
      <div className="dashboard-grid">
        <section className="panel wide">
          <div className="panel-heading"><div><h2>Recent profiles</h2><p>Start only after assigning an integrity-verified kernel.</p></div><button className="text-button" onClick={() => setView('profiles')}>View all →</button></div>
          {profileTable(bootstrap.profiles.slice(0, 5))}
        </section>
        <section className="panel rail-card">
          <div className="panel-heading"><div><h2>Safety posture</h2><p>Runtime boundaries that cannot silently weaken.</p></div></div>
          <ul className="check-list">
            <li><span>✓</span><div><strong>Verified kernels only</strong><p>Legacy executable paths stay dry-run only.</p></div></li>
            <li><span>✓</span><div><strong>Loopback-only CDP</strong><p>Readiness and WebSocket endpoints must stay local.</p></div></li>
            <li><span>✓</span><div><strong>OS-backed secrets</strong><p>Passwords are never written to profile or credential metadata.</p></div></li>
            <li><span>○</span><div><strong>Authenticated proxy bridge</strong><p>Credential-backed routes remain launch-blocked until the next phase.</p></div></li>
          </ul>
        </section>
      </div>
    </>
  }

  function renderProfiles() {
    return <>
      <div className="page-heading compact">
        <div><span className="eyebrow">Identity registry</span><h1>Browser profiles</h1><p>Start, stop, review, clone and edit locally isolated environments.</p></div>
        <button className="button primary" onClick={openCreate}>＋ New profile</button>
      </div>
      {runtimeError && <div className="form-error runtime-global-error">{runtimeError}</div>}
      <section className="panel">
        <div className="toolbar">
          <div className="search-box">⌕<input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search name, tag, kernel or proxy…" /></div>
          <select value={group} onChange={(event) => setGroup(event.target.value)}><option value="all">All groups</option>{groups.map((item) => <option key={item}>{item}</option>)}</select>
          <span className="result-count">{filtered.length} profile{filtered.length === 1 ? '' : 's'}</span>
        </div>
        {profileTable(filtered)}
      </section>
    </>
  }

  function renderRuntime() {
    return <>
      <div className="page-heading compact">
        <div><span className="eyebrow">Local process supervisor</span><h1>Runtime sessions</h1><p>Process state, loopback CDP readiness, log location and exit details are held in memory for the current desktop run.</p></div>
        <button className="button secondary" onClick={() => void refreshSessions()}>Refresh sessions</button>
      </div>
      {!backend.isNative() && <div className="info-banner runtime-mode-note"><strong>Desktop runtime required</strong><p>Browser preview mode can inspect the interface but cannot start local processes.</p></div>}
      {runtimeError && <div className="form-error runtime-global-error">{runtimeError}</div>}
      <RuntimePanel sessions={bootstrap.sessions} nativeMode={backend.isNative()} busyProfileID={runtimeBusyProfileID} onStop={(profileID) => void stopProfileByID(profileID)} />
    </>
  }

  function renderKernels() {
    const provider = bootstrap.providers.find((item) => item.id === kernelRequest.provider)
    return <>
      <div className="page-heading compact"><div><span className="eyebrow">Verified local binaries</span><h1>Kernel registry</h1><p>Import an existing Chromium executable into managed storage. Veilium records its provider contract and SHA-256 digest before runtime use.</p></div></div>
      <section className="panel kernel-import">
        <div className="panel-heading"><div><h2>Register local kernel</h2><p>Only regular files are accepted. Symbolic links are rejected.</p></div></div>
        <div className="kernel-import-grid">
          <label>Name<input value={kernelRequest.name} onChange={(event) => setKernelRequest((current) => ({ ...current, name: event.target.value }))} placeholder="Chromium 148 · Windows" /></label>
          <label>Provider<select value={kernelRequest.provider} onChange={(event) => { const next = bootstrap.providers.find((item) => item.id === event.target.value); setKernelRequest((current) => ({ ...current, provider: event.target.value, version: next?.versions[0] || '' })) }}>{bootstrap.providers.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <label>Version<select value={kernelRequest.version} onChange={(event) => setKernelRequest((current) => ({ ...current, version: event.target.value }))}>{provider?.versions.map((version) => <option key={version}>{version}</option>)}</select></label>
          <label className="kernel-path">Executable path<div className="path-picker"><input readOnly value={kernelRequest.sourcePath} placeholder="Choose a local Chromium executable…" /><button type="button" className="button secondary" onClick={() => void chooseKernel()} disabled={!backend.isNative()}>Browse</button></div></label>
          <button className="button primary kernel-import-button" onClick={() => void importKernel()} disabled={kernelBusy || !backend.isNative() || !kernelRequest.name.trim() || !kernelRequest.sourcePath.trim()}>{kernelBusy ? 'Working…' : 'Import and hash'}</button>
        </div>
        {!backend.isNative() && <div className="info-banner"><strong>Desktop runtime required</strong><p>Browser preview mode cannot read local files. Run Veilium through Wails to import a kernel.</p></div>}
        {kernelError && <div className="form-error">{kernelError}</div>}
      </section>
      <div className="kernel-registry-list">
        {bootstrap.kernels.length === 0
          ? <div className="panel empty-state"><div className="empty-icon">⬡</div><h3>No registered kernels</h3><p>Import a local Chromium executable to create the first managed record.</p></div>
          : bootstrap.kernels.map((record) => <article className="kernel-record" key={record.id}>
            <div className="kernel-record-head"><div className="kernel-symbol">⬡</div><div><h2>{record.name}</h2><code>{record.provider} · Chromium {record.version.split('.')[0]}</code></div><span className={`kernel-status ${record.status}`}>{record.status}</span></div>
            <dl><div><dt>SHA-256</dt><dd title={record.sha256}>{record.sha256.slice(0, 16)}…{record.sha256.slice(-8)}</dd></div><div><dt>Size</dt><dd>{(record.sizeBytes / 1024 / 1024).toFixed(1)} MB</dd></div><div><dt>Managed path</dt><dd title={record.executable}>{record.executable}</dd></div></dl>
            <div className="kernel-actions"><button className="button secondary" onClick={() => void verifyKernel(record)} disabled={kernelBusy}>Verify now</button><button className="button secondary danger-text" onClick={() => void deleteKernel(record)} disabled={kernelBusy}>Remove</button></div>
          </article>)}
      </div>
      <div className="provider-contracts"><h2>Provider contracts</h2><div className="kernel-grid">{bootstrap.providers.map((item) => <article className="kernel-card" key={item.id}><div className="kernel-title"><div className="kernel-symbol">⬡</div><div><h2>{item.name}</h2><code>{item.id}</code></div></div><p>{item.description}</p><div className="version-row">{item.versions.map((version) => <span key={version}>v{version.split('.')[0]}</span>)}</div></article>)}</div></div>
    </>
  }

  function renderCredentials() {
    return <>
      <div className="page-heading compact"><div><span className="eyebrow">Operating-system secret storage</span><h1>Credential vault</h1><p>Veilium stores only names, usernames and reference IDs. Password values stay inside {bootstrap.credentialProvider}.</p></div></div>
      <CredentialVault records={bootstrap.credentials} provider={bootstrap.credentialProvider} nativeMode={backend.isNative()} onSave={saveCredential} onDelete={deleteCredential} />
    </>
  }

  function renderSettings() {
    return <>
      <div className="page-heading compact"><div><span className="eyebrow">Application controls</span><h1>Settings</h1><p>Sensitive network bridges and automation capabilities remain gated.</p></div></div>
      <section className="settings-grid">
        <article className="panel setting-card"><h2>Runtime</h2><dl><div><dt>Application version</dt><dd>{bootstrap.version}</dd></div><div><dt>Frontend mode</dt><dd>{backend.isNative() ? 'Wails desktop' : 'Browser preview'}</dd></div><div><dt>Active sessions</dt><dd>{bootstrap.sessions.filter((item) => isRuntimeActive(item)).length}</dd></div><div><dt>Registered kernels</dt><dd>{bootstrap.kernels.length}</dd></div></dl></article>
        <article className="panel setting-card"><h2>Credential storage</h2><dl><div><dt>Provider</dt><dd>{bootstrap.credentialProvider}</dd></div><div><dt>Stored references</dt><dd>{bootstrap.credentials.length}</dd></div><div><dt>Plaintext fallback</dt><dd>Disabled</dd></div></dl></article>
        <article className="panel setting-card"><h2>Deferred security work</h2><ul className="plain-list"><li>Authenticated proxy bridge</li><li>Signed remote kernel manifests</li><li>Encrypted profile export/import</li><li>Real Chromium runtime matrix</li></ul></article>
      </section>
    </>
  }

  const activeEditingSession = editing ? sessionForProfile(bootstrap.sessions, editing.id) : undefined

  return (
    <div className="app-shell">
      <Sidebar active={view} onChange={setView} nativeMode={backend.isNative()} />
      <main className="main-content">
        <div className="topbar"><div className="window-context"><span className="context-dot" />Veilium workspace</div><div className="top-actions"><span className="version-chip">v{bootstrap.version}</span><button title="Refresh" onClick={() => void refresh()}>↻</button></div></div>
        <div className="page-content">
          {loading && <div className="loading-screen">Loading isolated identities…</div>}
          {error && <div className="form-error">{error}</div>}
          {!loading && view === 'dashboard' && renderDashboard()}
          {!loading && view === 'profiles' && renderProfiles()}
          {!loading && view === 'runtime' && renderRuntime()}
          {!loading && view === 'kernels' && renderKernels()}
          {!loading && view === 'credentials' && renderCredentials()}
          {!loading && view === 'settings' && renderSettings()}
        </div>
      </main>
      <ProfileEditor open={editorOpen && !isRuntimeActive(activeEditingSession)} profile={editing} providers={bootstrap.providers} kernels={bootstrap.kernels} credentials={bootstrap.credentials} onClose={() => setEditorOpen(false)} onSave={saveProfile} />
      <PlanDrawer profile={planProfile} plan={plan} error={planError} onClose={() => { setPlanProfile(undefined); setPlan(undefined) }} />
    </div>
  )
}
