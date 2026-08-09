import {
  SEVERITY_ORDER,
  SEVERITY_META,
  STATUS_ORDER,
  STATUS_META,
  CATEGORY_COLOR,
  CATEGORY_ORDER,
} from '../data/constants'

function Row({ label, count, total, color, active, onClick, bar }) {
  const pct = total ? Math.round((count / total) * 100) : 0
  return (
    <button
      type="button"
      className={active ? 'stat-row clickable active' : 'stat-row clickable'}
      style={{ '--cc': color }}
      onClick={onClick}
      title={`Filter by ${label}`}
    >
      {bar ? (
        <>
          <span className="stat-label">
            <span className="dot" style={{ background: color }} />
            {label}
          </span>
          <span className="bar">
            <i style={{ width: `${pct}%` }} />
          </span>
          <span className="num">{count}</span>
        </>
      ) : (
        <>
          <span className="stat-label">
            <span className="dot" style={{ background: color }} />
            {label}
          </span>
          <span className="bar bar-fill">
            <i style={{ width: `${pct}%`, background: color }} />
          </span>
          <span className="num">{count}</span>
        </>
      )}
    </button>
  )
}

export default function StatsSidebar({
  statusCounts,
  catCounts,
  sevCounts,
  total,
  status,
  category,
  severity,
  onStatusChange,
  onCategoryChange,
  onSeverityChange,
}) {
  return (
    <aside className="stats">
      <h3>Breakdown</h3>

      <div className="stat-group">
        <h4>Coverage</h4>
        {STATUS_ORDER.map((s) => (
          <Row
            key={s}
            label={STATUS_META[s].label}
            count={statusCounts[s] ?? 0}
            total={total}
            color={STATUS_META[s].color}
            active={status === s}
            onClick={() => onStatusChange(status === s ? 'all' : s)}
          />
        ))}
      </div>

      <div className="stat-group">
        <h4>Area</h4>
        {CATEGORY_ORDER.filter((c) => catCounts[c]).map((c) => (
          <Row
            key={c}
            label={c}
            count={catCounts[c]}
            total={total}
            color={CATEGORY_COLOR[c]}
            active={category === c}
            onClick={() => onCategoryChange(category === c ? 'all' : c)}
            bar
          />
        ))}
      </div>

      <div className="stat-group">
        <h4>Severity</h4>
        {SEVERITY_ORDER.map((s) => (
          <Row
            key={s}
            label={s}
            count={sevCounts[s] ?? 0}
            total={total}
            color={SEVERITY_META[s].bg}
            active={severity === s}
            onClick={() => onSeverityChange(severity === s ? 'all' : s)}
          />
        ))}
      </div>
    </aside>
  )
}
