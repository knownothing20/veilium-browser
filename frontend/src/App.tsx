import { useEffect, useMemo, useState } from 'react'
import { MetricCard } from './components/MetricCard'
import { PlanDrawer } from './components/PlanDrawer'
import { ProfileEditor } from './components/ProfileEditor'
import { ProfileTable } from './components/ProfileTable'
import { Sidebar, type ViewKey } from './components/Sidebar'
import { backend } from './lib/backend'
import { filterProfiles, groupsOf, profileHealth } from './lib/model'
import type { Bootstrap, LaunchPlan, Profile } from './types'

export default function App() {
  const [bootstrap, setBootstrap] = useState<Bootstrap>({ version: 'loading', profiles: [], providers: [] })
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

  useEffect(() => { void refresh() }, [])

  const filtered = useMemo(() => filterProfiles(bootstrap.profiles, query, group), [bootstrap.profiles, query, group])
  const groups = useMemo(() => groupsOf(bootstrap.profiles), [bootstrap.profiles])
  const metrics = useMemo(() => {
    const ready = bootstrap.profiles.filter((item) => profileHealth(item) === 'ready').length
    const routed = bootstrap.profiles.filter((item) => item.proxy.url && item.proxy.url !== 'direct://').length
    return { total: bootstrap.profiles.length, ready, routed, warnings: bootstrap.profiles.length - ready }
  }, [bootstrap.profiles])

  async function saveProfile(profile: Profile) {
    if (profile.id) await backend.updateProfile(profile)
    else await backend.createProfile(profile)
    await refresh()
  }

  async function cloneProfile(profile: Profile) {
    await backend.cloneProfile(profile.id, `${profile.name} Copy`)
    await refresh()
  }

  async function deleteProfile(profile: Profile) {
    if (!window.confirm(`Delete “${profile.name}”? Browser data is not removed in Phase 2.`)) return
    await backend.deleteProfile(profile.id)
    if (selected?.id === profile.id) setSelected(undefined)
    await refresh()
  }

  async function showPlan(profile: Profile) {
    setPlanProfile(profile)
    setPlan(undefined)
    setPlanError('')
    try {
      setPlan(await backend.buildLaunchPlan(profile.id))
    } catch (reason) {
      setPlanError(reason instanceof Error ? reason.message : String(reason))
    }
  }

  function openCreate() {
    setEditing(undefined)
    setEditorOpen(true)
  }

  function renderDashboard() {
    return <>
      <div className="page-heading">
        <div><span className="eyebrow">Local identity workspace</span><h1>Browser environments, without the guesswork.</h1><p>Every profile is tied to an explicit kernel contract, isolated data directory and reviewable network route.</p></div>
        <button className="button primary" onClick={openCreate}>＋ New profile</button>
      </div>
      <div className="metric-grid">
        <MetricCard label="Profiles" value={metrics.total} detail="Isolated local identities" />
        <MetricCard label="Ready" value={metrics.ready} detail="Passed visible configuration checks" tone="good" />
        <MetricCard label="Proxied" value={metrics.routed} detail="Profiles with explicit routes" />
        <MetricCard label="Needs review" value={metrics.warnings} detail="Incomplete or warning states" tone={metrics.warnings ? 'warn' : 'good'} />
      </div>
      <div className="dashboard-grid">
        <section className="panel wide">
          <div className="panel-heading"><div><h2>Recent profiles</h2><p>Configuration status before any browser process is launched.</p></div><button className="text-button" onClick={() => setView('profiles')}>View all →</button></div>
          <ProfileTable profiles={bootstrap.profiles.slice(0, 5)} selectedID={selected?.id} onSelect={setSelected} onEdit={(item) => { setEditing(item); setEditorOpen(true) }} onClone={(item) => void cloneProfile(item)} onPlan={(item) => void showPlan(item)} onDelete={(item) => void deleteProfile(item)} />
        </section>
        <section className="panel rail-card">
          <div className="panel-heading"><div><h2>Safety posture</h2><p>Defaults that cannot silently weaken.</p></div></div>
          <ul className="check-list">
            <li><span>✓</span><div><strong>Loopback-only CDP</strong><p>Remote debugging binds to 127.0.0.1.</p></div></li>
            <li><span>✓</span><div><strong>No inline proxy secrets</strong><p>Profiles persist credential references only.</p></div></li>
            <li><span>✓</span><div><strong>Version-aware controls</strong><p>Unsupported fingerprint flags stay disabled.</p></div></li>
            <li><span>○</span><div><strong>Process supervision</strong><p>Planned for Phase 3 after kernel verification.</p></div></li>
          </ul>
        </section>
      </div>
    </>
  }

  function renderProfiles() {
    return <>
      <div className="page-heading compact">
        <div><span className="eyebrow">Identity registry</span><h1>Browser profiles</h1><p>Search, review, clone and edit locally isolated environments.</p></div>
        <button className="button primary" onClick={openCreate}>＋ New profile</button>
      </div>
      <section className="panel">
        <div className="toolbar">
          <div className="search-box">⌕<input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search name, tag, kernel or proxy…" /></div>
          <select value={group} onChange={(event) => setGroup(event.target.value)}><option value="all">All groups</option>{groups.map((item) => <option key={item}>{item}</option>)}</select>
          <span className="result-count">{filtered.length} profile{filtered.length === 1 ? '' : 's'}</span>
        </div>
        <ProfileTable profiles={filtered} selectedID={selected?.id} onSelect={setSelected} onEdit={(item) => { setEditing(item); setEditorOpen(true) }} onClone={(item) => void cloneProfile(item)} onPlan={(item) => void showPlan(item)} onDelete={(item) => void deleteProfile(item)} />
      </section>
    </>
  }

  function renderKernels() {
    return <>
      <div className="page-heading compact"><div><span className="eyebrow">Capability contracts</span><h1>Kernel registry</h1><p>Veilium shows only controls verified for the selected Chromium provider and major version.</p></div></div>
      <div className="kernel-grid">
        {bootstrap.providers.map((provider) => <article className="kernel-card" key={provider.id}>
          <div className="kernel-title"><div className="kernel-symbol">⬡</div><div><h2>{provider.name}</h2><code>{provider.id}</code></div></div>
          <p>{provider.description}</p>
          <div className="version-row">{provider.versions.map((version) => <span key={version}>v{version.split('.')[0]}</span>)}</div>
          <div className="capability-table">
            {provider.samples.slice(0, 1).map((sample) => Object.entries({
              'Platform override': sample.canSetPlatform,
              'Brand override': sample.canSetBrand,
              'Timezone override': sample.canSetTimezone,
              'Seeded surfaces': sample.canSeedSurfaces,
              'Disable surfaces': sample.canDisableSurfaces,
              'Custom GPU': sample.canSetCustomGpu,
            }).map(([label, enabled]) => <div key={label}><span>{label}</span><strong className={enabled ? 'yes' : 'no'}>{enabled ? 'Verified' : 'Unavailable'}</strong></div>))}
          </div>
        </article>)}
      </div>
      <div className="info-banner"><strong>Why this matters</strong><p>Ant Browser demonstrated the value of a flexible external-kernel manager, but a UI can expose arguments that newer Chromium builds no longer accept. Veilium treats the provider/version capability table as a contract, not a hint.</p></div>
    </>
  }

  function renderSettings() {
    return <>
      <div className="page-heading compact"><div><span className="eyebrow">Application controls</span><h1>Settings</h1><p>Phase 2 exposes read-only runtime details while sensitive capabilities remain gated.</p></div></div>
      <section className="settings-grid">
        <article className="panel setting-card"><h2>Runtime</h2><dl><div><dt>Application version</dt><dd>{bootstrap.version}</dd></div><div><dt>Frontend mode</dt><dd>{backend.isNative() ? 'Wails desktop' : 'Browser preview'}</dd></div><div><dt>Storage</dt><dd>{backend.isNative() ? 'User config directory' : 'In-memory demo data'}</dd></div></dl></article>
        <article className="panel setting-card"><h2>Deferred security work</h2><ul className="plain-list"><li>OS credential vault integration</li><li>Signed kernel manifest verification</li><li>Process supervision and crash recovery</li><li>Encrypted profile export/import</li></ul></article>
        <article className="panel setting-card"><h2>Local API</h2><p>The existing bearer-authenticated REST service remains a separate command. Desktop bindings communicate inside the Wails runtime and do not expose another network listener.</p></article>
      </section>
    </>
  }

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
          {!loading && view === 'kernels' && renderKernels()}
          {!loading && view === 'settings' && renderSettings()}
        </div>
      </main>
      <ProfileEditor open={editorOpen} profile={editing} providers={bootstrap.providers} onClose={() => setEditorOpen(false)} onSave={saveProfile} />
      <PlanDrawer profile={planProfile} plan={plan} error={planError} onClose={() => { setPlanProfile(undefined); setPlan(undefined) }} />
    </div>
  )
}
