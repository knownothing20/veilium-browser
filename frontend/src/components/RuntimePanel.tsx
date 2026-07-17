import { isRuntimeActive } from '../lib/runtime'
import type { RuntimeSession } from '../types'

export function RuntimePanel({
  sessions,
  nativeMode,
  busyProfileID,
  onStop,
}: {
  sessions: RuntimeSession[]
  nativeMode: boolean
  busyProfileID?: string
  onStop: (profileId: string) => void
}) {
  if (sessions.length === 0) {
    return (
      <section className="panel empty-state">
        <div className="empty-icon">▶</div>
        <h3>No runtime sessions yet</h3>
        <p>Start a profile from Browser profiles after assigning a registered kernel.</p>
      </section>
    )
  }

  return (
    <div className="runtime-grid">
      {sessions.map((session) => {
        const active = isRuntimeActive(session)
        return (
          <article className="runtime-card" key={`${session.profileId}-${session.startedAt}`}>
            <div className="runtime-card-head">
              <div>
                <span className="eyebrow">{session.profileId.slice(0, 10)}</span>
                <h2>{session.profileName}</h2>
              </div>
              <span className={`runtime-state ${session.state}`}>{session.state}</span>
            </div>
            <dl>
              <div><dt>Process</dt><dd>{session.pid > 0 ? `PID ${session.pid}` : 'Not started'}</dd></div>
              <div><dt>CDP</dt><dd>{session.cdpUrl || 'Waiting for loopback endpoint'}</dd></div>
              <div><dt>Browser</dt><dd>{session.browser || 'Waiting for /json/version'}</dd></div>
              <div><dt>Started</dt><dd>{new Date(session.startedAt).toLocaleString()}</dd></div>
              <div><dt>Log</dt><dd title={session.logPath}>{session.logPath}</dd></div>
            </dl>
            {session.webSocketDebuggerUrl && <div className="runtime-endpoint"><span>WebSocket debugger</span><code>{session.webSocketDebuggerUrl}</code></div>}
            {session.lastError && <div className="runtime-failure"><strong>Runtime error</strong><p>{session.lastError}</p></div>}
            <div className="runtime-card-actions">
              {active && <button className="button secondary" disabled={!nativeMode || busyProfileID === session.profileId} onClick={() => onStop(session.profileId)}>{busyProfileID === session.profileId ? 'Stopping…' : 'Stop browser'}</button>}
              {!active && session.exitCode !== undefined && <span>Exit code {session.exitCode}</span>}
            </div>
          </article>
        )
      })}
    </div>
  )
}
