import RichText from '../RichText'

export default function CardsBlock({ block }) {
  return (
    <section>
      {block.heading && (
        <h2>
          <RichText>{block.heading}</RichText>
        </h2>
      )}
      <div className="bento">
        {block.items.map((it, i) => (
          <div className="bento-card" key={i}>
            <h3>
              <RichText>{it.title}</RichText>
            </h3>
            <p>
              <RichText>{it.body}</RichText>
            </p>
          </div>
        ))}
      </div>
    </section>
  )
}
