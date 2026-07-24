import { healthLabel, formatDateTime } from '../i18n/format'
import { isRuntimeActive } from '../lib/runtime'
import type { RuntimeSession } from '../types'
import { AppIcon } from './AppIcon'

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
        <div className="empty-icon"><AppIcon name="runtime" size={25} /></div>
        <h3>还没有运行会话</h3>
        <p>请先在“浏览器环境”中为环境分配已注册内核，然后点击“打开浏览器”。</p>
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
              <span className={`runtime-state ${session.state}`}>{healthLabel(session.state)}</span>
            </div>
            <dl>
              <div><dt>进程</dt><dd>{session.pid > 0 ? `PID ${session.pid}` : '尚未启动'}</dd></div>
              <div><dt>CDP 端点</dt><dd>{session.cdpUrl || '正在等待本机回环端点'}</dd></div>
              <div><dt>浏览器信息</dt><dd>{session.browser || '正在等待 /json/version'}</dd></div>
              <div><dt>启动时间</dt><dd>{formatDateTime(session.startedAt)}</dd></div>
              <div><dt>日志路径</dt><dd title={session.logPath}>{session.logPath}</dd></div>
            </dl>
            {session.webSocketDebuggerUrl && <div className="runtime-endpoint"><span>WebSocket 调试端点</span><code>{session.webSocketDebuggerUrl}</code></div>}
            {session.lastError && <div className="runtime-failure"><strong>运行错误</strong><p>{session.lastError}</p></div>}
            <div className="runtime-card-actions">
              {active && <button className="button secondary" disabled={!nativeMode || busyProfileID === session.profileId} onClick={() => onStop(session.profileId)}>{busyProfileID === session.profileId ? '正在关闭…' : '关闭浏览器'}</button>}
              {!active && session.exitCode !== undefined && <span>退出代码 {session.exitCode}</span>}
            </div>
          </article>
        )
      })}
    </div>
  )
}
