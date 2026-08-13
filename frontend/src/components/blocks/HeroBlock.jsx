import RichText from '../RichText'

export default function HeroBlock({ block }) {
  return (
    <div className="hero">
      <h1>
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
