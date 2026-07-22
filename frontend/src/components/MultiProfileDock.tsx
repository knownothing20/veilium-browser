import { useEffect, useState } from 'react'
import { normalizeLifecycleBootstrap, type LifecycleBootstrap } from '../lifecycle'
import { backend } from '../lib/backend'
import { MultiProfileWorkspace } from './MultiProfileWorkspace'

const emptyData: LifecycleBootstrap = normalizeLifecycleBootstrap({
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

export function MultiProfileDock() {
  const [open, setOpen] = useState(false)
  const [data, setData] = useState<LifecycleBootstrap>(emptyData)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const refresh = async () => {
    setLoading(true)
    try {
      setData(normalizeLifecycleBootstrap(await backend.bootstrap()))
      setError('')
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (open) void refresh()
  }, [open])

  return <>
    <button className="multi-profile-dock-button" onClick={() => setOpen((value) => !value)} aria-expanded={open}>
      {open ? 'Close Phase 5 tools' : 'Multi-Profile tools'}
    </button>
    {open && <aside className="multi-profile-dock" aria-label="Multi-Profile and storage tools">
      <div className="multi-profile-dock-header">
        <div><span className="eyebrow">Phase 5 workspace</span><h1>Multi-Profile tools</h1><p>Bounded metadata changes and read-only storage inventory.</p></div>
        <button className="button secondary" disabled={loading} onClick={() => void refresh()}>{loading ? 'Refreshing…' : 'Refresh data'}</button>
      </div>
      {error && <div className="form-error">{error}</div>}
      <MultiProfileWorkspace data={data} onRefresh={refresh} />
    </aside>}
  </>
}
