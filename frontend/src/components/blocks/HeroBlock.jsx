import RichText from '../RichText'
import { slugify } from './slugify'

export default function HeroBlock({ block }) {
  return (
    <div className="hero">
      <h1 id={slugify(block.title)}>
        <RichText>{block.title}</RichText>
      </h1>
      {block.lede && (
        <p className="lede">
          <RichText>{block.lede}</RichText>
        </p>
      )}
    </div>
  )
}
