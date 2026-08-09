import { STATUS_META, CATEGORY_COLOR } from '../data/constants'

function Chip({ label, count, color, active, onClick, dot = true, title }) {
  return (
    <button
      type="button"
      className={active ? 'chip active' : 'chip'}
      style={{ '--cc': color }}
      onClick={onClick}
      title={title}
    >
      {dot && <span className="dot" style={{ background: color }} />}
      {label}
      {count !== undefined && <span className="chip-n">{count}</span>}
    </button>
  )
}

export default function FilterChips({
  status,
  statusCounts,
  onStatusChange,
  category,
  categories,
  catCounts,
  onCategoryChange,
}) {
  return (
    <>
      <nav className="chips status-row" aria-label="Filter by coverage status">
        <Chip
          label="All statuses"
          color="#B9B5AA"
          active={status === 'all'}
          onClick={() => onStatusChange('all')}
          dot={false}
        />
        {Object.entries(STATUS_META).map(([key, meta]) => (
          <Chip
            key={key}
            label={meta.label}
            color={meta.color}
            count={statusCounts[key] ?? 0}
            active={status === key}
            onClick={() => onStatusChange(key)}
          />
        ))}
      </nav>

      <nav className="chips" aria-label="Filter by area">
        <Chip
          label="All"
          color="#B9B5AA"
          count={undefined}
          active={category === 'all'}
          onClick={() => onCategoryChange('all')}
        />
        {categories.map((c) => (
          <Chip
            key={c}
            label={c}
            color={CATEGORY_COLOR[c] ?? '#B9B5AA'}
            count={catCounts[c] ?? 0}
            active={category === c}
            onClick={() => onCategoryChange(c)}
          />
        ))}
      </nav>
    </>
  )
}
