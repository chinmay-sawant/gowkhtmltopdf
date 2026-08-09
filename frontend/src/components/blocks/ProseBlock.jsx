export default function ProseBlock({ block }) {
  return (
    <section className="prose">
      {block.heading && <h2>{block.heading}</h2>}
      {block.sections.map((s, i) => (
        <div className="prose-item" key={i}>
          {s.heading && <h3>{s.heading}</h3>}
          {s.body && <p>{s.body}</p>}
          {s.bullets && (
            <ul className="bullets">
              {s.bullets.map((b, j) => (
                <li key={j}>{b}</li>
              ))}
            </ul>
          )}
        </div>
      ))}
    </section>
  )
}
