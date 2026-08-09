export default function StatsBlock({ block }) {
  return (
    <div className="statband">
      {block.items.map((it, i) => (
        <div className="statband-item" key={i}>
          <div className={it.percent ? 'statband-value has-percent' : 'statband-value'}>
            {it.value}
            {it.percent && <span className="statband-percent">{it.percent}</span>}
          </div>
          <div className="statband-label">{it.label}</div>
        </div>
      ))}
    </div>
  )
}
