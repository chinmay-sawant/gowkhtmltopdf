import HeroBlock from './HeroBlock'
import StatsBlock from './StatsBlock'
import CardsBlock from './CardsBlock'
import ProseBlock from './ProseBlock'
import CodeBlock from './CodeBlock'
import TableBlock from './TableBlock'
import BulletsBlock from './BulletsBlock'
import CalloutBlock from './CalloutBlock'

const RENDERERS = {
  hero: HeroBlock,
  stats: StatsBlock,
  cards: CardsBlock,
  prose: ProseBlock,
  code: CodeBlock,
  table: TableBlock,
  bullets: BulletsBlock,
  callout: CalloutBlock,
}

export default function ContentBlocks({ content }) {
  return (
    <div className="content-blocks">
      {content.map((block, i) => {
        const Cmp = RENDERERS[block.type]
        if (!Cmp) return null
        return <Cmp block={block} key={i} />
      })}
    </div>
  )
}
