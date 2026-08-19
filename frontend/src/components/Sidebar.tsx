import { ui } from '../i18n'
import { AppIcon, type AppIconName } from './AppIcon'

export type ViewKey =
  | 'environments'
  | 'network'
  | 'recovery'
  | 'batch'
  | 'settings'
  | 'runtime'
  | 'kernels'
  | 'credentials'

const primaryNavigation: Array<{ key: ViewKey; label: string; icon: AppIconName }> = [
  { key: 'environments', label: ui.nav.environments, icon: 'environment' },
  { key: 'network', label: ui.nav.network, icon: 'network' },
  { key: 'recovery', label: ui.nav.recovery, icon: 'recovery' },
  { key: 'batch', label: ui.nav.batch, icon: 'batch' },
  { key: 'settings', label: ui.nav.settings, icon: 'settings' },
]

const advancedNavigation: Array<{ key: ViewKey; label: string; icon: AppIconName }> = [
  { key: 'runtime', label: ui.nav.runtime, icon: 'runtime' },
  { key: 'kernels', label: ui.nav.kernels, icon: 'kernel' },
  { key: 'credentials', label: ui.nav.credentials, icon: 'credential' },
]

export function Sidebar({ active, onChange, nativeMode }: { active: ViewKey; onChange: (view: ViewKey) => void; nativeMode: boolean }) {
  const renderItem = (item: { key: ViewKey; label: string; icon: AppIconName }) => (
    <button
      className={active === item.key ? 'nav-item active' : 'nav-item'}
      key={item.key}
      onClick={() => onChange(item.key)}
      aria-current={active === item.key ? 'page' : undefined}
    >
      <span className="nav-icon"><AppIcon name={item.icon} /></span>
      <span>{item.label}</span>
    </button>
  )

  return (
    <aside className="sidebar">
      <button className="brand brand-button" onClick={() => onChange('environments')} aria-label={ui.nav.environments}>
        <div className="brand-mark">V</div>
        <div><strong>Veilium</strong><span>{ui.app.subtitle}</span></div>
      </button>
      <div className="nav-section-label">{ui.nav.primary}</div>
      <nav aria-label={ui.nav.primary}>{primaryNavigation.map(renderItem)}</nav>
      <div className="nav-section-label advanced">{ui.nav.advanced}</div>
      <nav aria-label={ui.nav.advanced}>{advancedNavigation.map(renderItem)}</nav>
      <div className="sidebar-footer">
        <div className={nativeMode ? 'mode-dot online' : 'mode-dot'} />
        <div>
          <strong>{nativeMode ? ui.app.desktopRuntime : ui.app.browserPreview}</strong>
          <span>{nativeMode ? ui.app.desktopEnabled : ui.app.previewDisabled}</span>
        </div>
      </div>
    </aside>
  )
}
