export default function CardsBlock({ block }) {
  return (
    <section>
      {block.heading && <h2>{block.heading}</h2>}
      <div className="bento">
        {block.items.map((it, i) => (
          <div className="bento-card" key={i}>
            <h3>{it.title}</h3>
            <p>{it.body}</p>
          </div>
        ))}
      </div>
    </section>
  )
}
