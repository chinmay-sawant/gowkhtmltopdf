const ICON = {
  info: 'i',
  warn: '!',
  tip: 't',
}

const LABEL = {
  info: 'Scope note',
  warn: 'Boundary to know',
  tip: 'Practical tip',
}

export default function CalloutBlock({ block }) {
  const variant = block.variant || 'info'

  return (
    <aside className={`callout callout-${variant}`} role="note">
      <div className="callout-marker" aria-hidden="true">
        {ICON[variant] || 'i'}
      </div>
      <div className="callout-body">
        <span className="callout-kicker">{LABEL[variant] || LABEL.info}</span>
        {block.title && <h3 className="callout-title">{block.title}</h3>}
        {block.body && <p>{block.body}</p>}
      </div>
    </aside>
  )
}
