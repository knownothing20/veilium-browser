import { lifecycleAllowsEdit, lifecycleAllowsLaunch, lifecycleRecordFor, type LifecycleRecord } from '../lifecycle'
import { healthLabel, lifecycleStateLabel } from '../i18n/format'
import { ui } from '../i18n'
import { profileHealth } from '../lib/model'
import { isRuntimeActive, runtimeStateLabel, sessionForProfile } from '../lib/runtime'
import type { Profile, RuntimeSession } from '../types'
import { AppIcon } from './AppIcon'
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
  emptyKind = 'search',
  onCreate,
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
  emptyKind?: 'first-use' | 'search'
  onCreate?: () => void
  onSelect: (profile: Profile) => void
  onEdit: (profile: Profile) => void
  onClone: (profile: Profile) => void
  onPlan: (profile: Profile) => void
  onStart: (profile: Profile) => void
  onStop: (profile: Profile) => void
  onDelete: (profile: Profile) => void
}) {
  if (profiles.length === 0) {
    const firstUse = emptyKind === 'first-use'
    return (
      <div className="empty-state environment-empty-state">
        <div className="empty-icon"><AppIcon name="environment" size={25} /></div>
        <h3>{firstUse ? ui.environments.emptyTitle : ui.environments.noMatchTitle}</h3>
        <p>{firstUse ? ui.environments.emptyDetail : ui.environments.noMatchDetail}</p>
        {firstUse && onCreate && <button className="button primary empty-primary-action" onClick={onCreate}><AppIcon name="add" />{ui.common.create}</button>}
      </div>
    )
  }

  return (
    <div className="table-wrap">
      <table className="profile-table environment-table">
        <thead>
          <tr>
            <th>{ui.environments.identity}</th>
            <th>{ui.environments.browser}</th>
            <th>{ui.environments.route}</th>
            <th>{ui.environments.identityConfig}</th>
            <th>{ui.environments.status}</th>
            <th aria-label={ui.environments.actions}>{ui.environments.actions}</th>
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
            const trashAllowed = nativeMode && !active && !lifecycle?.lock && Boolean(lifecycle && ['available', 'draft', 'archived'].includes(lifecycle.state))
            const lifecycleReason = lifecycle?.lock?.operationId || lifecycle?.limitationCodes?.join(' · ') || lifecycle?.recoveryCodes?.join(' · ')
            const lifecycleClass = lifecycle?.state || 'missing'
            const launchDisabled = !nativeMode || !profile.kernel.id || !launchAllowed || busyProfileID === profile.id
            const launchTitle = !nativeMode
              ? ui.environments.startRequiresDesktop
              : !profile.kernel.id || !launchAllowed
                ? ui.environments.startBlocked
                : ui.environments.openBrowser
            return (
              <tr className={selectedID === profile.id ? 'selected' : ''} key={profile.id} onClick={() => onSelect(profile)}>
                <td>
                  <div className="identity-cell">
                    <div className="avatar">{profile.name.slice(0, 2).toUpperCase()}</div>
                    <div>
                      <strong>{profile.name}</strong>
                      <span>{profile.group || ui.common.defaultGroup} · {(profile.tags || []).join(' · ') || ui.common.noTags}</span>
                    </div>
                  </div>
                </td>
                <td>
                  <strong>{kernelProviderLabel(profile.kernel.provider)}</strong>
                  <span>Chromium {profile.kernel.version.split('.')[0]} · {profile.kernel.id ? '已注册' : '旧版路径'}</span>
                </td>
                <td>
                  <strong>{profile.proxy.url === 'direct://' ? '直连' : '代理'}</strong>
                  <span title={profile.proxy.url || 'direct://'}>{profile.proxy.url || 'direct://'}</span>
                </td>
                <td>
                  <strong>{platformLabel(profile.fingerprint.platform)} · {profile.fingerprint.language}</strong>
                  <span>{profile.fingerprint.timezone}</span>
                </td>
                <td>
                  <div className="lifecycle-status-stack">
                    <span className={`status-pill ${active ? 'running' : health}`}>{active ? healthLabel(runtimeStateLabel(session?.state)) : healthLabel(health)}</span>
                    <span className={`lifecycle-pill ${lifecycleClass} ${lifecycle?.lock ? 'locked' : ''}`}>{lifecycleStateLabel(lifecycle?.state, Boolean(lifecycle?.lock))}</span>
                    {lifecycleReason && <span className="lifecycle-reason" title={lifecycleReason}>{lifecycleReason}</span>}
                    {session?.state === 'failed' && <span className="runtime-error-inline" title={session.lastError}>{session.lastError || '浏览器启动失败'}</span>}
                  </div>
                </td>
                <td>
                  <div className="row-actions environment-actions" onClick={(event) => event.stopPropagation()}>
                    {active
                      ? <button className="button compact stop-button" title={ui.environments.closeBrowser} disabled={busyProfileID === profile.id} onClick={() => onStop(profile)}><AppIcon name="stop" />{ui.environments.closeBrowser}</button>
                      : <button className="button compact primary launch-button" title={launchTitle} disabled={launchDisabled} onClick={() => onStart(profile)}><AppIcon name="launch" />{ui.environments.openBrowser}</button>}
                    <button className="button compact secondary" title={active ? ui.environments.stopBeforeEdit : ui.common.edit} disabled={active || !editAllowed} onClick={() => onEdit(profile)}><AppIcon name="edit" />{ui.common.edit}</button>
                    <details className="row-more">
                      <summary className="button compact secondary"><AppIcon name="more" />{ui.common.more}</summary>
                      <div className="row-more-menu">
                        <div className="technical-actions" aria-label={ui.common.details}>
                          <ProxyDiagnosticAction profile={profile} nativeMode={nativeMode && launchAllowed} />
                          <EvidenceAction profile={profile} session={session} nativeMode={nativeMode && launchAllowed} />
                          <NetworkEvidenceAction profile={profile} session={session} nativeMode={nativeMode && launchAllowed} />
                          <ConsistencyAction profile={profile} nativeMode={nativeMode && launchAllowed} />
                        </div>
                        <button disabled={!launchAllowed} onClick={() => onPlan(profile)}>查看启动详情</button>
                        <button disabled={!launchAllowed} onClick={() => onClone(profile)}>{ui.common.clone}</button>
                        <button className="danger-text" title={trashAllowed ? ui.environments.moveToTrash : ui.environments.startBlocked} disabled={!trashAllowed} onClick={() => onDelete(profile)}>{ui.environments.moveToTrash}</button>
                      </div>
                    </details>
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

function kernelProviderLabel(provider: string): string {
  if (provider === 'patched-chromium') return '已审查 Chromium'
  if (provider === 'custom-chromium') return '自定义 Chromium'
  return '本机 Chromium'
}

function platformLabel(platform: string): string {
  if (platform === 'windows') return 'Windows'
  if (platform === 'linux') return 'Linux'
  if (platform === 'macos') return 'macOS'
  return platform
}
