export default function BulletsBlock({ block }) {
  return (
    <section className="prose">
      {block.heading && <h2>{block.heading}</h2>}
      <ul className="bullets">
        {block.items.map((item, i) => (
          <li key={i}>{item}</li>
        ))}
      </ul>
    </section>
  )
}
