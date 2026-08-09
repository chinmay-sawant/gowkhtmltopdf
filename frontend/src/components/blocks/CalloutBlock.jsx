const ICON = {
  info: 'i',
  warn: '!',
  tip: 't',
}

export default function CalloutBlock({ block }) {
  return (
    <aside className={`callout callout-${block.variant || 'info'}`} role="note">
      <div className="callout-marker">{ICON[block.variant] || 'i'}</div>
      <div className="callout-body">
        {block.title && <strong>{block.title}</strong>}
        {block.body && <p>{block.body}</p>}
      </div>
    </aside>
  )
}
