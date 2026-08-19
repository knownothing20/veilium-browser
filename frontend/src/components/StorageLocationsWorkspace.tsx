import { useEffect, useState } from 'react'
import { formatDateTime } from '../i18n/format'
import {
  storageLocationAPI,
  type ManagedStorageLocation,
  type ManagedStorageLocations,
} from '../storageLocations'

export function StorageLocationsWorkspace() {
  const nativeMode = storageLocationAPI.isNative()
  const [data, setData] = useState<ManagedStorageLocations>()
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState('')
  const [error, setError] = useState('')

  const refresh = async () => {
    setLoading(true)
    setError('')
    try { setData(await storageLocationAPI.get()) }
    catch (reason) { setError(reason instanceof Error ? reason.message : String(reason)) }
    finally { setLoading(false) }
  }

  useEffect(() => {
    if (nativeMode) void refresh()
  }, [nativeMode])

  const copyPath = async (item: ManagedStorageLocation) => {
    try {
      await navigator.clipboard.writeText(item.path)
      setCopied(item.id)
      window.setTimeout(() => setCopied((current) => current === item.id ? '' : current), 1600)
    } catch (reason) {
      setError(`复制路径失败：${reason instanceof Error ? reason.message : String(reason)}`)
    }
  }

  return <section className="panel recovery-section">
    <div className="panel-heading">
      <div><span className="eyebrow">仅限本机的受管路径</span><h2>存储位置</h2><p>查看 Veilium 保存环境、浏览器内核、代理组件、日志、生命周期状态、快照、可恢复回收站和模板的精确位置。此页面不会移动或删除数据。</p></div>
      <button className="button secondary" disabled={!nativeMode || loading} onClick={() => void refresh()}>{loading ? '正在刷新…' : '刷新存储位置'}</button>
    </div>

    {!nativeMode && <div className="form-error">查看受管存储位置需要 Wails 桌面运行时。</div>}
    {error && <div className="form-error">{error}</div>}

    {data && <>
      <div className={data.onSystemVolume ? 'form-error' : 'info-banner'}>
        <strong>{data.onSystemVolume ? 'Veilium 数据位于 Windows 系统盘' : 'Veilium 数据根目录不在检测到的 Windows 系统盘'}</strong>
        <p>{data.onSystemVolume
          ? '环境浏览器数据和受管浏览器包可能占用大量空间。安装大型组件前，请检查下面的路径。'
          : '下列路径只属于本机安装。便携定义和操作报告不会包含这些绝对路径。'}</p>
      </div>
      <dl>
        <div><dt>数据根目录</dt><dd><code>{data.dataRoot}</code></dd></div>
        <div><dt>数据所在卷</dt><dd>{data.dataVolume || '未知'}</dd></div>
        <div><dt>Windows 系统卷</dt><dd>{data.systemVolume || '当前平台未检测到'}</dd></div>
        <div><dt>检查时间</dt><dd>{formatDateTime(data.generatedAt)}</dd></div>
      </dl>

      <div className="recovery-card-grid">
        {(data.locations || []).map((item) => <article className="recovery-card" key={item.id}>
          <div className="recovery-card-head"><strong>{locationLabel(item)}</strong><span className={`lifecycle-operation-status ${statusClass(item)}`}>{locationStatusLabel(item)}</span></div>
          <p>{locationDescription(item)}</p>
          <code title={item.path}>{item.path}</code>
          <dl>
            <div><dt>预期类型</dt><dd>{kindLabel(item.kind)}</dd></div>
            <div><dt>所在卷</dt><dd>{item.volume || '未知'}</dd></div>
            <div><dt>系统卷</dt><dd>{item.onSystemVolume ? '是' : '否'}</dd></div>
            {item.reasonCode && <div><dt>检查结果</dt><dd>{item.reasonCode}</dd></div>}
          </dl>
          <button className="button secondary" disabled={loading} onClick={() => void copyPath(item)}>{copied === item.id ? '已复制' : '复制路径'}</button>
        </article>)}
      </div>

      <div className="info-banner"><strong>安全边界</strong><ul className="plain-list">{(data.limitations || []).map((item) => <li key={item}>{item}</li>)}</ul></div>
    </>}
  </section>
}

function statusClass(item: ManagedStorageLocation): string {
  if (item.status === 'present') return 'passed'
  if (item.status === 'missing') return 'running'
  return 'failed'
}
function locationStatusLabel(item: ManagedStorageLocation): string {
  if (item.status === 'present') return '存在'
  if (item.status === 'missing') return '尚未创建'
  if (item.status === 'unsafe-link') return '不安全链接'
  if (item.status === 'unexpected-entry') return '非预期条目'
  return '不可用'
}
function kindLabel(value: string): string { const labels: Record<string, string> = { directory: '目录', file: '文件' }; return labels[value] || value }
function locationLabel(item: ManagedStorageLocation): string {
  const labels: Record<string, string> = { profiles: '浏览器环境数据', kernels: '浏览器内核', adapters: '代理组件', runtime: '运行时数据', logs: '运行日志', lifecycle: '生命周期状态', snapshots: '本地快照', trash: '可恢复回收站', templates: '环境模板' }
  return labels[item.id] || item.label
}
function locationDescription(item: ManagedStorageLocation): string {
  const descriptions: Record<string, string> = {
    profiles: '每个浏览器环境独立的受管用户数据目录。',
    kernels: '经过导入、哈希和完整性记录的浏览器内核。',
    adapters: '经过导入、哈希和验证的 Xray 与 sing-box 运行组件。',
    runtime: '每次启动产生的受限临时运行材料。',
    logs: '本机浏览器与代理运行日志。',
    lifecycle: '生命周期记录、操作日志、锁和恢复状态。',
    snapshots: '经过验证的同机环境快照。',
    trash: '可恢复环境数据及其保留记录。',
    templates: '不包含秘密与浏览器数据的私有环境模板。',
  }
  return descriptions[item.id] || item.description
}
