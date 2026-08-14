import { useMemo, useState, useRef, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import PageTitle from '../components/PageTitle'
import Cardbox from '../components/Cardbox'
import {
  SHOWCASE,
  SHOWCASE_SPECIAL,
  SHOWCASE_CATEGORIES,
  SHOWCASE_CATEGORY_COLORS,
} from '../data/showcase'

const THUMBS = import.meta.glob('../assets/showcase/thumbs/*.webp', { eager: true, query: '?url', import: 'default' })
const IMAGES = import.meta.glob('../assets/showcase/*.png', { eager: true, query: '?url', import: 'default' })

function thumbUrl(item, page) {
  const name = page === 1 ? item.name : `${item.name}-${page}`
  return THUMBS[`../assets/showcase/thumbs/${name}.webp`] || IMAGES[`../assets/showcase/${name}.png`]
}

function ShowcaseCard({ item, index, onOpen, page, onPageChange }) {
  const total = item.pages ?? 1
  const multi = total > 1
  const src = thumbUrl(item, page)

  const handleCardClick = (e) => {
    // If click was inside stepper, don't open modal
    if (e.target.closest('.showcase-stepper')) return
    onOpen(item)
  }

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      if (e.target.closest('.showcase-stepper')) return
      e.preventDefault()
      onOpen(item)
    }
  }

  return (
    <article
      className="showcase-card"
      style={{ '--index': index }}
      onClick={handleCardClick}
      onKeyDown={handleKeyDown}
      tabIndex={0}
      role="button"
      aria-label={`Open sample ${item.title}`}
    >
      <div className="showcase-thumb">
        {src ? (
          <img
            src={src}
            alt={`${item.title} page ${page}`}
            loading={index < 4 ? 'eager' : 'lazy'}
            width="400"
            height="566"
          />
        ) : null}
        {multi && (
          <div className="showcase-stepper" onClick={(e) => e.stopPropagation()}>
            <button
              type="button"
              className="stepper-btn"
              aria-label="Previous page"
              title="Previous page"
              onClick={(e) => {
                e.stopPropagation()
                onPageChange(page > 1 ? page - 1 : total)
              }}
            >
              ←
            </button>
            <span className="stepper-count" aria-label={`Page ${page} of ${total}`}>
              {page}/{total}
            </span>
            <button
              type="button"
              className="stepper-btn"
              aria-label="Next page"
              title="Next page"
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
        {item.category && (
          <span
            className="showcase-category-tag"
            style={{ '--cc': SHOWCASE_CATEGORY_COLORS[item.category] ?? '#B9B5AA' }}
          >
            <span className="dot" style={{ background: SHOWCASE_CATEGORY_COLORS[item.category] ?? '#B9B5AA' }} />
            {item.category}
          </span>
        )}
        <h3>{item.title}</h3>
        <p>{item.desc}</p>
        <span className="showcase-file">{item.file}</span>
        <button
          type="button"
          className="showcase-open"
          onClick={(e) => {
            e.stopPropagation()
            onOpen(item)
          }}
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
  const [searchParams, setSearchParams] = useSearchParams()
  const chipNavRef = useRef(null)

  const allItems = useMemo(() => [...SHOWCASE, ...SHOWCASE_SPECIAL], [])
  const allCategories = useMemo(() => ['All', ...SHOWCASE_CATEGORIES], [])
  const activeCat = searchParams.get('cat') || 'All'

  const filteredItems = useMemo(() => {
    if (activeCat === 'All') return allItems
    return allItems.filter((item) => item.category === activeCat)
  }, [allItems, activeCat])

  const handleCategoryChange = useCallback(
    (cat) => {
      if (cat === 'All') {
        const nextParams = new URLSearchParams(searchParams)
        nextParams.delete('cat')
        setSearchParams(nextParams, { replace: true })
      } else {
        const nextParams = new URLSearchParams(searchParams)
        nextParams.set('cat', cat)
        setSearchParams(nextParams, { replace: true })
      }
    },
    [searchParams, setSearchParams],
  )

  // Keyboard navigation for category filter chips
  const handleChipKeyDown = (e, catIndex) => {
    let nextIndex = null
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
      e.preventDefault()
      nextIndex = (catIndex + 1) % allCategories.length
    } else if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') {
      e.preventDefault()
      nextIndex = (catIndex - 1 + allCategories.length) % allCategories.length
    } else if (e.key === 'Home') {
      e.preventDefault()
      nextIndex = 0
    } else if (e.key === 'End') {
      e.preventDefault()
      nextIndex = allCategories.length - 1
    }

    if (nextIndex !== null) {
      const nextCategory = allCategories[nextIndex]
      handleCategoryChange(nextCategory)
      const buttons = chipNavRef.current?.querySelectorAll('button.chip')
      if (buttons && buttons[nextIndex]) {
        buttons[nextIndex].focus()
      }
    }
  }

  const setPage = (name) => (p) => setPages((prev) => ({ ...prev, [name]: p }))

  return (
    <>
      <PageTitle title="Showcase" />
      <section className="showcase-hero" aria-labelledby="showcase-title">
        <div>
          <h1 id="showcase-title">Showcase</h1>
          <p className="lede">
            Explore committed PDFs generated from golden HTML fixtures. Multi-page samples can be stepped
            through in place, then opened in the full viewer.
          </p>
        </div>
        <div className="showcase-meta">
          <span>{filteredItems.length} of {allItems.length} samples</span>
          <span>{activeCat === 'All' ? 'all categories' : activeCat}</span>
          <span>PDF output you can inspect</span>
        </div>
      </section>

      <div className="showcase-controls" aria-label="Category filters">
        <nav
          ref={chipNavRef}
          className="chips showcase-chips"
          aria-label="Filter showcase by category"
          role="tablist"
        >
          <button
            type="button"
            className={activeCat === 'All' ? 'chip active' : 'chip'}
            style={{ '--cc': '#B9B5AA' }}
            onClick={() => handleCategoryChange('All')}
            onKeyDown={(e) => handleChipKeyDown(e, 0)}
            role="tab"
            aria-selected={activeCat === 'All'}
            aria-label={`All categories (${allItems.length} samples)`}
          >
            All
            <span className="chip-n">{allItems.length}</span>
          </button>
          {SHOWCASE_CATEGORIES.map((cat, idx) => {
            const count = allItems.filter((it) => it.category === cat).length
            const color = SHOWCASE_CATEGORY_COLORS[cat] ?? '#B9B5AA'
            const isCatActive = activeCat === cat
            return (
              <button
                type="button"
                key={cat}
                className={isCatActive ? 'chip active' : 'chip'}
                style={{ '--cc': color }}
                onClick={() => handleCategoryChange(cat)}
                onKeyDown={(e) => handleChipKeyDown(e, idx + 1)}
                role="tab"
                aria-selected={isCatActive}
                aria-label={`${cat} category (${count} samples)`}
              >
                <span className="dot" style={{ background: color }} />
                {cat}
                <span className="chip-n">{count}</span>
              </button>
            )
          })}
        </nav>
      </div>

      <div className="showcase-grid" role="region" aria-label="Showcase PDF samples">
        {filteredItems.map((item, i) => (
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


