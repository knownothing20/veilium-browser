const lifecycleLabels: Record<string, string> = {
  available: '可用',
  draft: '待完善',
  archived: '已归档',
  trashed: '回收站',
  invalid: '无效',
  missing: '缺少生命周期记录',
}

const healthLabels: Record<string, string> = {
  ready: '可启动',
  warning: '需要检查',
  incomplete: '配置未完成',
  running: '运行中',
  failed: '启动失败',
  starting: '正在启动',
  stopping: '正在关闭',
  stopped: '已停止',
  exited: '已退出',
}

const statusLabels: Record<string, string> = {
  verified: '已验证',
  available: '可用',
  installed: '已安装',
  invalid: '无效',
  missing: '缺失',
  modified: '已修改',
  unverified: '未验证',
}

export function healthLabel(value?: string): string {
  if (!value) return '未知状态'
  return healthLabels[value] || value
}

export function lifecycleStateLabel(value?: string, locked = false): string {
  const label = lifecycleLabels[value || 'missing'] || value || lifecycleLabels.missing
  return locked ? `${label} · 已锁定` : label
}

export function statusLabel(value?: string): string {
  if (!value) return '未知'
  return statusLabels[value] || value
}

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  const value = bytes / 1024 ** index
  return `${value >= 10 || index === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[index]}`
}

export function formatDateTime(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}
