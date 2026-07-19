import { profileHealth } from '../lib/model'
import { isRuntimeActive, runtimeStateLabel, sessionForProfile } from '../lib/runtime'
import type { Profile, RuntimeSession } from '../types'
import { ConsistencyAction } from './ConsistencyAction'
import { EvidenceAction } from './EvidenceAction'
import { ProxyDiagnosticAction } from './ProxyDiagnosticAction'

export function ProfileTable({
  profiles,
  sessions,
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
                  <span className={`status-pill ${active ? 'running' : health}`}>{active ? runtimeStateLabel(session?.state) : health}</span>
                  {session?.state === 'failed' && <span className="runtime-error-inline" title={session.lastError}>{session.lastError || 'Runtime failed'}</span>}
                </td>
                <td>
                  <div className="row-actions" onClick={(event) => event.stopPropagation()}>
                    {active
                      ? <button className="stop-icon" title="Stop browser" disabled={busyProfileID === profile.id} onClick={() => onStop(profile)}>■</button>
                      : <button title={nativeMode ? 'Start browser' : 'Desktop runtime required'} disabled={!nativeMode || !profile.kernel.id || busyProfileID === profile.id} onClick={() => onStart(profile)}>▶</button>}
                    <ProxyDiagnosticAction profile={profile} nativeMode={nativeMode} />
                    <EvidenceAction profile={profile} session={session} nativeMode={nativeMode} />
                    <ConsistencyAction profile={profile} nativeMode={nativeMode} />
                    <button title="Review launch plan" onClick={() => onPlan(profile)}>≡</button>
                    <button title={active ? 'Stop browser before editing' : 'Edit'} disabled={active} onClick={() => onEdit(profile)}>✎</button>
                    <button title="Clone" onClick={() => onClone(profile)}>⧉</button>
                    <button className="danger-icon" title={active ? 'Stop browser before deleting' : 'Delete'} disabled={active} onClick={() => onDelete(profile)}>×</button>
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
