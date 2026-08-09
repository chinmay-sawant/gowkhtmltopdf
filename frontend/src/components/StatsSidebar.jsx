import {
  SEVERITY_ORDER,
  SEVERITY_META,
  STATUS_ORDER,
  STATUS_META,
  CATEGORY_COLOR,
  CATEGORY_ORDER,
} from '../data/constants'

function BarRow({ label, count, total, color }) {
  const pct = total ? Math.round((count / total) * 100) : 0
  return (
    <div className="stat-row">
      <div>
        <span className="dot" style={{ background: color }} />
        {label}
      </div>
      <div className="bar">
        <i style={{ width: `${pct}%` }} />
      </div>
      <div className="num">
        {count}
        <small>/{total}</small>
      </div>
    </div>
  )
}

export default function StatsSidebar({
  statusCounts,
  catCounts,
  sevCounts,
  total,
  status,
  onStatusChange,
}) {
  return (
    <aside className="stats">
      <h3>Breakdown</h3>
      <div className="stat-group">
        <h4>Coverage in gowkhtmltopdf</h4>
        {STATUS_ORDER.map((s) => {
          const active = status === s
          return (
            <button
              type="button"
              className={active ? 'sev-row clickable active' : 'sev-row clickable'}
              key={s}
              data-status={s}
              style={{ '--cc': STATUS_META[s].color }}
              onClick={() => onStatusChange(active ? 'all' : s)}
            >
              <span>
                <span className="dot" style={{ background: STATUS_META[s].color }} />{' '}
                {STATUS_META[s].label}
              </span>
              <span className="num">{statusCounts[s] ?? 0}</span>
            </button>
          )
        })}
      </div>
      <div className="stat-group">
        <h4>Area</h4>
        {CATEGORY_ORDER.filter((c) => catCounts[c]).map((c) => (
          <BarRow key={c} label={c} count={catCounts[c]} total={total} color={CATEGORY_COLOR[c]} />
        ))}
      </div>
      <div className="stat-group">
        <h4>Severity</h4>
        {SEVERITY_ORDER.map((s) => (
          <div className="sev-row" key={s}>
            <span>
              <span className="dot" style={{ background: SEVERITY_META[s].bg }} /> {s}
            </span>
            <span className="num">{sevCounts[s] ?? 0}</span>
          </div>
        ))}
      </div>
    </aside>
  )
}
