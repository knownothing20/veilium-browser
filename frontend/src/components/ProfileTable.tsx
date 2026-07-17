import { profileHealth } from '../lib/model'
import type { Profile } from '../types'

export function ProfileTable({
  profiles,
  selectedID,
  onSelect,
  onEdit,
  onClone,
  onPlan,
  onDelete,
}: {
  profiles: Profile[]
  selectedID?: string
  onSelect: (profile: Profile) => void
  onEdit: (profile: Profile) => void
  onClone: (profile: Profile) => void
  onPlan: (profile: Profile) => void
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
                <td>
                  <strong>{profile.kernel.provider === 'patched-chromium' ? 'Patched' : 'Native'}</strong>
                  <span>Chromium {profile.kernel.version.split('.')[0]}</span>
                </td>
                <td>
                  <strong>{profile.proxy.url === 'direct://' ? 'Direct' : 'Proxy'}</strong>
                  <span>{profile.proxy.url || 'direct://'}</span>
                </td>
                <td>
                  <strong>{profile.fingerprint.platform} · {profile.fingerprint.language}</strong>
                  <span>{profile.fingerprint.timezone}</span>
                </td>
                <td><span className={`status-pill ${health}`}>{health}</span></td>
                <td>
                  <div className="row-actions" onClick={(event) => event.stopPropagation()}>
                    <button title="Launch plan" onClick={() => onPlan(profile)}>▶</button>
                    <button title="Edit" onClick={() => onEdit(profile)}>✎</button>
                    <button title="Clone" onClick={() => onClone(profile)}>⧉</button>
                    <button className="danger-icon" title="Delete" onClick={() => onDelete(profile)}>×</button>
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
