import type { LaunchPlan, Profile } from '../types'

export function PlanDrawer({ profile, plan, error, onClose }: { profile?: Profile; plan?: LaunchPlan; error?: string; onClose: () => void }) {
  if (!profile) return null
  return <div className="overlay" onMouseDown={onClose}>
    <aside className="plan-drawer" onMouseDown={(event) => event.stopPropagation()}>
      <header className="editor-header">
        <div><span className="eyebrow">启动前检查</span><h2>启动详情</h2><p>{profile.name}</p></div>
        <button className="close-button" onClick={onClose} aria-label="关闭">×</button>
      </header>
      {error && <div className="form-error">{error}</div>}
      {!plan && !error && <div className="loading-block">正在生成确定性的启动计划…</div>}
      {plan && <div className="plan-content">
        <div className="plan-summary"><span>可执行文件</span><code>{plan.executable}</code></div>
        <div className="plan-summary"><span>代理路由</span><code>{plan.proxyDisplay}</code></div>
        <div className="plan-summary"><span>本地代理桥</span><strong>{plan.requiresBridge ? plan.bridgeKind || '需要' : '不需要'}</strong></div>
        {plan.warnings && plan.warnings.length > 0 && <div className="warning-box"><strong>打开浏览器前请检查</strong>{plan.warnings.map((warning) => <p key={warning}>{warning}</p>)}</div>}
        <h3>启动参数</h3>
        <pre>{plan.args.join('\n')}</pre>
        <p className="drawer-note">这里展示 Veilium 将使用的精确参数。实际启动仍要求已注册并通过完整性验证的浏览器内核、受管环境目录、仅绑定本机回环的 CDP，以及可用的代理桥。</p>
      </div>}
    </aside>
  </div>
}
