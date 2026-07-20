import { lifecycleAllowsEdit, lifecycleAllowsLaunch, lifecycleLabel, lifecycleRecordFor, type LifecycleRecord } from '../lifecycle'
import { profileHealth } from '../lib/model'
import { isRuntimeActive, runtimeStateLabel, sessionForProfile } from '../lib/runtime'
import type { Profile, RuntimeSession } from '../types'
import { ConsistencyAction } from './ConsistencyAction'
import { EvidenceAction } from './EvidenceAction'
import { NetworkEvidenceAction } from './NetworkEvidenceAction'
import { ProxyDiagnosticAction } from './ProxyDiagnosticAction'

export function ProfileTable({
  profiles,
  sessions,
  lifecycleRecords,
  selectedID,
  nativeMode,
  busyProfileID,
  onSelect,
  onEdit,
  onClone,
  onPlan,
  onStart,
  onStop,
  onDelete,
}: {
  profiles: Profile[]
  sessions: RuntimeSession[]
  lifecycleRecords: LifecycleRecord[]
  selectedID?: string
  nativeMode: boolean
  busyProfileID?: string
  onSelect: (profile: Profile) => void
  onEdit: (profile: Profile) => void
  onClone: (profile: Profile) => void
  onPlan: (profile: Profile) => void
  onStart: (profile: Profile) => void
  onStop: (profile: Profile) => void
  onDelete: (profile: Profile) => void
}) {
  if (profiles.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-icon">◎</div>
        <h3>No matching profiles</h3>
        <p>Create a profile or adjust the search filters.</p>
      </div>
    )
  }

  return (
    <div className="table-wrap">
      <table className="profile-table">
        <thead>
          <tr>
            <th>Identity</th>
            <th>Kernel</th>
            <th>Route</th>
            <th>Fingerprint</th>
            <th>Status</th>
            <th aria-label="Actions" />
          </tr>
        </thead>
        <tbody>
          {profiles.map((profile) => {
            const health = profileHealth(profile)
            const session = sessionForProfile(sessions, profile.id)
            const active = isRuntimeActive(session)
            const lifecycle = lifecycleRecordFor(lifecycleRecords, profile.id)
            const launchAllowed = lifecycleAllowsLaunch(lifecycle)
            const editAllowed = lifecycleAllowsEdit(lifecycle)
            const lifecycleReason = lifecycle?.lock?.operationId || lifecycle?.limitationCodes?.join(' · ') || lifecycle?.recoveryCodes?.join(' · ')
            const lifecycleClass = lifecycle?.state || 'missing'
            return (
              <tr className={selectedID === profile.id ? 'selected' : ''} key={profile.id} onClick={() => onSelect(profile)}>
                <td>
                  <div className="identity-cell">
                    <div className="avatar">{profile.name.slice(0, 2).toUpperCase()}</div>
                    <div>
                      <strong>{profile.name}</strong>
                      <span>{profile.group || 'Default'} · {(profile.tags || []).join(' · ') || 'No tags'}</span>
                    </div>
                  </div>
                </td>
                <td><strong>{profile.kernel.provider === 'patched-chromium' ? 'Patched' : profile.kernel.provider === 'custom-chromium' ? 'Custom' : 'Native'}</strong><span>Chromium {profile.kernel.version.split('.')[0]} · {profile.kernel.id ? 'registered' : 'legacy'}</span></td>
                <td><strong>{profile.proxy.url === 'direct://' ? 'Direct' : 'Proxy'}</strong><span>{profile.proxy.url || 'direct://'}</span></td>
                <td><strong>{profile.fingerprint.platform} · {profile.fingerprint.language}</strong><span>{profile.fingerprint.timezone}</span></td>
                <td>
                  <div className="lifecycle-status-stack">
                    <span className={`status-pill ${active ? 'running' : health}`}>{active ? runtimeStateLabel(session?.state) : health}</span>
                    <span className={`lifecycle-pill ${lifecycleClass} ${lifecycle?.lock ? 'locked' : ''}`}>{lifecycleLabel(lifecycle)}</span>
                    {lifecycleReason && <span className="lifecycle-reason" title={lifecycleReason}>{lifecycleReason}</span>}
                    {session?.state === 'failed' && <span className="runtime-error-inline" title={session.lastError}>{session.lastError || 'Runtime failed'}</span>}
                  </div>
                </td>
                <td>
                  <div className="row-actions" onClick={(event) => event.stopPropagation()}>
                    {active
                      ? <button className="stop-icon" title="Stop browser" disabled={busyProfileID === profile.id} onClick={() => onStop(profile)}>■</button>
                      : <button title={!nativeMode ? 'Desktop runtime required' : !launchAllowed ? `Lifecycle state blocks launch: ${lifecycleLabel(lifecycle)}` : 'Start browser'} disabled={!nativeMode || !profile.kernel.id || !launchAllowed || busyProfileID === profile.id} onClick={() => onStart(profile)}>▶</button>}
                    <ProxyDiagnosticAction profile={profile} nativeMode={nativeMode && launchAllowed} />
                    <EvidenceAction profile={profile} session={session} nativeMode={nativeMode && launchAllowed} />
                    <NetworkEvidenceAction profile={profile} session={session} nativeMode={nativeMode && launchAllowed} />
                    <ConsistencyAction profile={profile} nativeMode={nativeMode && launchAllowed} />
                    <button title={launchAllowed ? 'Review launch plan' : `Lifecycle state blocks launch plan: ${lifecycleLabel(lifecycle)}`} disabled={!launchAllowed} onClick={() => onPlan(profile)}>≡</button>
                    <button title={active ? 'Stop browser before editing' : !editAllowed ? `Lifecycle state blocks editing: ${lifecycleLabel(lifecycle)}` : 'Edit'} disabled={active || !editAllowed} onClick={() => onEdit(profile)}>✎</button>
                    <button title={launchAllowed ? 'Clone' : `Lifecycle state blocks cloning: ${lifecycleLabel(lifecycle)}`} disabled={!launchAllowed} onClick={() => onClone(profile)}>⧉</button>
                    <button className="danger-icon" title="Trash operations are not available until M5.2" disabled onClick={() => onDelete(profile)}>×</button>
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
