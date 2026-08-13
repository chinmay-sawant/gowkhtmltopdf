import RichText from '../RichText'

export default function BulletsBlock({ block }) {
  return (
    <section className="prose">
      {block.heading && (
        <h2>
          <RichText>{block.heading}</RichText>
        </h2>
      )}
      <ul className="bullets">
        {block.items.map((item, i) => (
          <li key={i}>
            <RichText>{item}</RichText>
          </li>
        ))}
      </ul>
    </section>
  )
}
