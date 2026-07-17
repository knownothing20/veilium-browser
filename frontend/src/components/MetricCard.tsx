export function MetricCard({ label, value, detail, tone = 'neutral' }: { label: string; value: string | number; detail: string; tone?: 'neutral' | 'good' | 'warn' }) {
  return (
    <article className={`metric-card ${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </article>
  )
}
