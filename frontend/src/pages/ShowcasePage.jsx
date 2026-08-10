import { useState } from 'react'
import PageTitle from '../components/PageTitle'
import Cardbox from '../components/Cardbox'
import { SHOWCASE, SHOWCASE_SPECIAL } from '../data/showcase'

const IMAGES = import.meta.glob('../assets/showcase/*.png', { eager: true, query: '?url', import: 'default' })

function pageUrl(item, page) {
  if (page === 1) return IMAGES[`../assets/showcase/${item.name}.png`]
  return IMAGES[`../assets/showcase/${item.name}-${page}.png`]
}

function ShowcaseCard({ item, index, onOpen, page, onPageChange }) {
  const total = item.pages ?? 1
  const multi = total > 1
  const src = pageUrl(item, page)

  return (
    <article className="showcase-card" style={{ '--index': index }}>
      <div className="showcase-thumb">
        {src ? <img src={src} alt={`${item.title} page ${page}`} loading="lazy" /> : null}
        {multi && (
          <div className="showcase-stepper" onClick={(e) => e.stopPropagation()}>
            <button
              type="button"
              className="stepper-btn"
              aria-label="Previous page"
              onClick={(e) => {
                e.stopPropagation()
                onPageChange(page > 1 ? page - 1 : total)
              }}
            >
              ←
            </button>
            <span className="stepper-count">
              {page}/{total}
            </span>
            <button
              type="button"
              className="stepper-btn"
              aria-label="Next page"
              onClick={(e) => {
                e.stopPropagation()
                onPageChange(page < total ? page + 1 : 1)
              }}
            >
              →
            </button>
          </div>
        )}
      </div>
      <div className="showcase-body">
        <h3>{item.title}</h3>
        <p>{item.desc}</p>
        <span className="showcase-file">{item.file}</span>
        <button
          type="button"
          className="showcase-open"
          onClick={() => onOpen(item)}
          aria-label={`Open ${item.title} sample`}
        >
          Open sample <span aria-hidden="true">→</span>
        </button>
      </div>
    </article>
  )
}

export default function ShowcasePage() {
  const [active, setActive] = useState(null)
  const [pages, setPages] = useState({})

  const setPage = (name) => (p) => setPages((prev) => ({ ...prev, [name]: p }))

  return (
    <>
      <PageTitle title="Showcase" />
      <section className="showcase-hero" aria-labelledby="showcase-title">
        <h1 id="showcase-title">Showcase</h1>
        <p className="lede">
          Explore committed PDFs generated from golden HTML fixtures. Multi-page samples can be stepped
          through in place, then opened in the full viewer.
        </p>
        <div className="showcase-meta">
          <span>{SHOWCASE.length + SHOWCASE_SPECIAL.length} samples</span>
          <span>business documents and CSS fixtures</span>
          <span>PDF output you can inspect</span>
        </div>
      </section>

      <div className="showcase-grid">
        {[...SHOWCASE, ...SHOWCASE_SPECIAL].map((item, i) => (
          <ShowcaseCard
            key={item.name}
            item={item}
            index={i}
            onOpen={setActive}
            page={pages[item.name] ?? 1}
            onPageChange={setPage(item.name)}
          />
        ))}
      </div>

      {active && <Cardbox item={active} onClose={() => setActive(null)} />}
    </>
  )
}
