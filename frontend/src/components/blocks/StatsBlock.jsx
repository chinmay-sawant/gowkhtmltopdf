export default function StatsBlock({ block }) {
  return (
    <div className="statband">
      {block.items.map((it, i) => (
        <div className="statband-item" key={i}>
          <div className="statband-value">{it.value}</div>
          <div className="statband-label">{it.label}</div>
        </div>
      ))}
    </div>
  )
}
