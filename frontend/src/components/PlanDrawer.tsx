import type { LaunchPlan, Profile } from '../types'

export function PlanDrawer({ profile, plan, error, onClose }: { profile?: Profile; plan?: LaunchPlan; error?: string; onClose: () => void }) {
  if (!profile) return null
  return (
    <div className="overlay" onMouseDown={onClose}>
      <aside className="plan-drawer" onMouseDown={(event) => event.stopPropagation()}>
        <header className="editor-header">
          <div><span className="eyebrow">Dry run only</span><h2>Launch plan</h2><p>{profile.name}</p></div>
          <button className="close-button" onClick={onClose}>×</button>
        </header>
        {error && <div className="form-error">{error}</div>}
        {!plan && !error && <div className="loading-block">Generating a deterministic launch plan…</div>}
        {plan && <div className="plan-content">
          <div className="plan-summary"><span>Executable</span><code>{plan.executable}</code></div>
          <div className="plan-summary"><span>Proxy route</span><code>{plan.proxyDisplay}</code></div>
          <div className="plan-summary"><span>Bridge</span><strong>{plan.requiresBridge ? plan.bridgeKind || 'required' : 'not required'}</strong></div>
          {plan.warnings && plan.warnings.length > 0 && <div className="warning-box"><strong>Review before launch</strong>{plan.warnings.map((warning) => <p key={warning}>{warning}</p>)}</div>}
          <h3>Arguments</h3>
          <pre>{plan.args.join('\n')}</pre>
          <p className="drawer-note">Phase 2 produces and reviews launch plans. Process supervision is intentionally deferred until the kernel verification and credential-vault layers are complete.</p>
        </div>}
      </aside>
    </div>
  )
}
