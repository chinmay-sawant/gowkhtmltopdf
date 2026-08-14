import RichText from '../RichText'
import { slugify } from './slugify'

export default function ProseBlock({ block }) {
  return (
    <section className="prose">
      {block.heading && (
        <h2 id={slugify(block.heading)}>
          <RichText>{block.heading}</RichText>
        </h2>
      )}
      {block.sections.map((s, i) => (
        <div className="prose-item" key={i}>
          {s.heading && (
            <h3 id={slugify(s.heading)}>
              <RichText>{s.heading}</RichText>
            </h3>
          )}
          {s.body && (
            <p>
              <RichText>{s.body}</RichText>
            </p>
          )}
          {s.bullets && (
            <ul className="bullets">
              {s.bullets.map((b, j) => (
                <li key={j}>
                  <RichText>{b}</RichText>
                </li>
              ))}
            </ul>
          )}
        </div>
      ))}
    </section>
  )
}
