export type ViewKey = 'dashboard' | 'profiles' | 'runtime' | 'kernels' | 'settings'

const navigation: Array<{ key: ViewKey; label: string; icon: string }> = [
  { key: 'dashboard', label: 'Overview', icon: '◫' },
  { key: 'profiles', label: 'Browser profiles', icon: '◎' },
  { key: 'runtime', label: 'Runtime sessions', icon: '▶' },
  { key: 'kernels', label: 'Kernel registry', icon: '⬡' },
  { key: 'settings', label: 'Settings', icon: '⚙' },
]

export function Sidebar({ active, onChange, nativeMode }: { active: ViewKey; onChange: (view: ViewKey) => void; nativeMode: boolean }) {
  return (
    <aside className="sidebar">
      <div className="brand">
        <div className="brand-mark">V</div>
        <div>
          <strong>Veilium</strong>
          <span>Privacy browser</span>
        </div>
      </div>
      <nav>
        {navigation.map((item) => (
          <button className={active === item.key ? 'nav-item active' : 'nav-item'} key={item.key} onClick={() => onChange(item.key)}>
            <span className="nav-icon">{item.icon}</span>
            {item.label}
          </button>
        ))}
      </nav>
      <div className="sidebar-footer">
        <div className={nativeMode ? 'mode-dot online' : 'mode-dot'} />
        <div>
          <strong>{nativeMode ? 'Desktop runtime' : 'Browser preview'}</strong>
          <span>{nativeMode ? 'Local process control enabled' : 'Process execution disabled'}</span>
        </div>
      </div>
    </aside>
  )
}
