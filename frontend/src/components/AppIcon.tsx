export type AppIconName =
  | 'environment'
  | 'network'
  | 'recovery'
  | 'batch'
  | 'settings'
  | 'runtime'
  | 'kernel'
  | 'credential'
  | 'refresh'
  | 'search'
  | 'launch'
  | 'stop'
  | 'edit'
  | 'more'
  | 'add'

const paths: Record<AppIconName, React.ReactNode> = {
  environment: <><rect x="3" y="4" width="8" height="7" rx="2" /><rect x="13" y="4" width="8" height="7" rx="2" /><rect x="3" y="13" width="8" height="7" rx="2" /><rect x="13" y="13" width="8" height="7" rx="2" /></>,
  network: <><circle cx="5" cy="12" r="2.5" /><circle cx="19" cy="6" r="2.5" /><circle cx="19" cy="18" r="2.5" /><path d="M7.5 11 16.5 7M7.5 13 16.5 17" /></>,
  recovery: <><path d="M4 10a8 8 0 1 1 2.4 8.1" /><path d="M4 4v6h6" /></>,
  batch: <><rect x="4" y="4" width="12" height="12" rx="2" /><path d="M8 20h10a2 2 0 0 0 2-2V8" /></>,
  settings: <><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1-2.8 2.8-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.6v.2h-4V21a1.7 1.7 0 0 0-1-1.6 1.7 1.7 0 0 0-1.9.3l-.1.1L4.2 17l.1-.1a1.7 1.7 0 0 0 .3-1.9A1.7 1.7 0 0 0 3 14H2.8v-4H3a1.7 1.7 0 0 0 1.6-1 1.7 1.7 0 0 0-.3-1.9L4.2 7 7 4.2l.1.1A1.7 1.7 0 0 0 9 4.6 1.7 1.7 0 0 0 10 3V2.8h4V3a1.7 1.7 0 0 0 1 1.6 1.7 1.7 0 0 0 1.9-.3l.1-.1L19.8 7l-.1.1a1.7 1.7 0 0 0-.3 1.9 1.7 1.7 0 0 0 1.6 1h.2v4H21a1.7 1.7 0 0 0-1.6 1Z" /></>,
  runtime: <><path d="m8 5 10 7-10 7Z" /></>,
  kernel: <><path d="m12 3 8 4.5v9L12 21l-8-4.5v-9Z" /><circle cx="12" cy="12" r="3" /></>,
  credential: <><path d="M12 3 5 6v5c0 4.5 2.8 7.7 7 10 4.2-2.3 7-5.5 7-10V6Z" /><path d="M9 12h6M12 9v6" /></>,
  refresh: <><path d="M20 6v5h-5" /><path d="M4 18v-5h5" /><path d="M18.5 9A7 7 0 0 0 6.2 6.2L4 8M5.5 15A7 7 0 0 0 17.8 17.8L20 16" /></>,
  search: <><circle cx="11" cy="11" r="7" /><path d="m20 20-4-4" /></>,
  launch: <><path d="m9 5 10 7-10 7Z" /></>,
  stop: <><rect x="6" y="6" width="12" height="12" rx="2" /></>,
  edit: <><path d="M4 20h4l11-11-4-4L4 16Z" /><path d="m13.5 6.5 4 4" /></>,
  more: <><circle cx="5" cy="12" r="1" fill="currentColor" stroke="none" /><circle cx="12" cy="12" r="1" fill="currentColor" stroke="none" /><circle cx="19" cy="12" r="1" fill="currentColor" stroke="none" /></>,
  add: <><path d="M12 5v14M5 12h14" /></>,
}

export function AppIcon({ name, size = 18 }: { name: AppIconName; size?: number }) {
  return <svg className="app-icon" width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name]}</svg>
}
