export default function HeroBlock({ block }) {
  return (
    <div className="hero">
      <h1>{block.title}</h1>
      {block.lede && <p className="lede">{block.lede}</p>}
    </div>
  )
}
