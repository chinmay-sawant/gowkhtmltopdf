import RichText from '../RichText'
import { slugify } from './slugify'

export default function TocBlock({ block }) {
  const items = block.items || []

  return (
    <nav className="toc-block" aria-label={block.title || 'Table of contents'}>
      {block.title && <h2 className="toc-title">{block.title}</h2>}
      <ul className="toc-list">
        {items.map((item, i) => {
          const text = typeof item === 'string' ? item : item.label || item.title || ''
          const target = typeof item === 'object' && item.href ? item.href : `#${slugify(text)}`
          return (
            <li key={i} className="toc-item">
              <a href={target} className="toc-link">
                <RichText>{text}</RichText>
              </a>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
