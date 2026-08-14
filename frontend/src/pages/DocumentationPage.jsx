import { useMemo, useState, useEffect } from 'react'
import { useParams, NavLink, Link, Navigate } from 'react-router-dom'
import ContentBlocks from '../components/blocks/ContentBlocks'
import { slugify } from '../components/blocks/slugify'
import PageTitle from '../components/PageTitle'
import Footer from '../components/Footer'

function loadContent(id) {
  const pages = import.meta.glob('../data/content/page-*.json', { eager: true })
  for (const path in pages) {
    if (pages[path].id === id) return pages[path]
  }
  return null
}

const DOCS = ['cli', 'library-api', 'architecture', 'compatibility', 'fonts', 'security', 'performance']

const DOC_FILE_MAP = {
  cli: 'cli.md',
  'library-api': 'library-api.md',
  architecture: 'architecture.md',
  compatibility: 'compatibility-matrix.md',
  fonts: 'fonts.md',
  security: 'integration-security.md',
  performance: 'performance.md',
}

function calculateReadingTime(content) {
  if (!content) return 1
  let text = ''
  function traverse(node) {
    if (!node) return
    if (typeof node === 'string') {
      text += ' ' + node
    } else if (Array.isArray(node)) {
      node.forEach(traverse)
    } else if (typeof node === 'object') {
      Object.values(node).forEach(traverse)
    }
  }
  traverse(content)
  const words = text.trim().split(/\s+/).filter(Boolean).length
  return Math.max(1, Math.ceil(words / 200))
}

export default function DocumentationPage() {
  const { docId } = useParams()
  const [headings, setHeadings] = useState([])
  const [activeId, setActiveId] = useState('')

  const docs = useMemo(
    () =>
      DOCS.map((id) => {
        const p = loadContent(id)
        return { id, nav: p ? p.nav : id }
      }),
    [],
  )

  const page = useMemo(() => loadContent(docId), [docId])

  const readingTime = useMemo(() => {
    return page ? calculateReadingTime(page.content) : 1
  }, [page])

  const currentIndex = DOCS.indexOf(docId)
  const prevDoc = currentIndex > 0 ? docs[currentIndex - 1] : null
  const nextDoc = currentIndex >= 0 && currentIndex < docs.length - 1 ? docs[currentIndex + 1] : null

  const githubDocFile = DOC_FILE_MAP[docId] || `${docId}.md`
  const githubUrl = `https://github.com/chinmay-sawant/gowkhtmltopdf/blob/master/documentation/${githubDocFile}`

  // Scrollspy & heading discovery
  useEffect(() => {
    if (!page) return

    // Small delay to ensure ContentBlocks DOM elements are mounted
    const timer = setTimeout(() => {
      const headingEls = Array.from(
        document.querySelectorAll(
          '.docs-main .content-blocks h2, .docs-main .content-blocks h3, .docs-main .prose h2, .docs-main .prose h3, .docs-main .code-block-heading, .docs-main .table-block-heading, .docs-main .callout-title',
        ),
      )

      const items = []
      headingEls.forEach((el) => {
        if (!el.id) {
          el.id = slugify(el.textContent)
        }
        if (el.id && el.textContent.trim()) {
          items.push({
            id: el.id,
            text: el.textContent.replace(/^[#\s]+/, '').trim(),
            level: el.tagName.toLowerCase() === 'h3' ? 'h3' : 'h2',
          })
        }
      })

      setHeadings(items)
      if (items.length > 0 && !window.location.hash) {
        setActiveId(items[0].id)
      } else if (window.location.hash) {
        setActiveId(window.location.hash.replace('#', ''))
      }

      if (headingEls.length === 0) return

      const observer = new IntersectionObserver(
        (entries) => {
          const visible = entries.filter((e) => e.isIntersecting)
          if (visible.length > 0) {
            // Pick heading closest to top
            visible.sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top)
            setActiveId(visible[0].target.id)
          }
        },
        {
          rootMargin: '-80px 0px -70% 0px',
          threshold: [0, 0.5, 1],
        },
      )

      headingEls.forEach((el) => observer.observe(el))

      return () => {
        observer.disconnect()
      }
    }, 50)

    return () => clearTimeout(timer)
  }, [docId, page])

  if (!page) return <Navigate to="/documentation/cli" replace />

  const scrollToHeading = (id, e) => {
    if (e) e.preventDefault()
    const el = document.getElementById(id)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'start' })
      window.history.replaceState(null, '', `#${id}`)
      setActiveId(id)
    }
  }

  return (
    <div className="docs-page" id="main-content" tabIndex={-1}>
      <PageTitle title={page.nav} />
      <div className="docs-layout">
        {/* Left Navigation Sidebar */}
        <aside className="docs-sidebar" aria-label="Documentation Sidebar">
          <div className="docs-sidebar-head">Documentation</div>
          <nav className="docs-nav" aria-label="Documentation sections">
            {docs.map((d) => (
              <NavLink
                key={d.id}
                to={`/documentation/${d.id}`}
                className={({ isActive }) => (isActive ? 'docs-link active' : 'docs-link')}
              >
                {d.nav}
              </NavLink>
            ))}
          </nav>
        </aside>

        {/* Center Main Content */}
        <main className="docs-main" id="docs-article-body">
          <div className="docs-context" aria-label="Current documentation section">
            <div className="docs-breadcrumbs">
              <span>Documentation</span>
              <span aria-hidden="true" className="docs-breadcrumb-sep">
                /
              </span>
              <strong>{page.nav}</strong>
            </div>
            <div className="docs-reading-badge" title="Estimated reading time">
              <svg
                className="docs-badge-icon"
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <circle cx="12" cy="12" r="10" />
                <polyline points="12 6 12 12 16 14" />
              </svg>
              <span>{readingTime} min read</span>
            </div>
          </div>

          <ContentBlocks content={page.content} />

          {/* Documentation Footer Actions & Pagination */}
          <div className="docs-footer-actions">
            <a
              href={githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="docs-edit-github-link"
              title="Edit this documentation page on GitHub"
            >
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
              </svg>
              <span>Edit this page on GitHub</span>
            </a>
          </div>

          {/* Previous / Next Article Navigation Cards */}
          <nav className="docs-pagination" aria-label="Adjacent documentation pages">
            {prevDoc ? (
              <Link
                to={`/documentation/${prevDoc.id}`}
                className="docs-pagination-card docs-pagination-prev"
                title={`Go to previous section: ${prevDoc.nav}`}
              >
                <span className="docs-pagination-sub">
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    aria-hidden="true"
                  >
                    <polyline points="15 18 9 12 15 6" />
                  </svg>
                  Previous Article
                </span>
                <span className="docs-pagination-title">{prevDoc.nav}</span>
              </Link>
            ) : (
              <div className="docs-pagination-spacer" />
            )}

            {nextDoc ? (
              <Link
                to={`/documentation/${nextDoc.id}`}
                className="docs-pagination-card docs-pagination-next"
                title={`Go to next section: ${nextDoc.nav}`}
              >
                <span className="docs-pagination-sub">
                  Next Article
                  <svg
                    width="14"
                    height="14"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2.5"
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    aria-hidden="true"
                  >
                    <polyline points="9 18 15 12 9 6" />
                  </svg>
                </span>
                <span className="docs-pagination-title">{nextDoc.nav}</span>
              </Link>
            ) : (
              <div className="docs-pagination-spacer" />
            )}
          </nav>
        </main>

        {/* Right Sticky Scrollspy TOC */}
        <aside className="docs-toc" aria-label="Table of contents">
          <div className="docs-toc-inner">
            <div className="docs-toc-head">On this page</div>
            {headings.length > 0 ? (
              <nav className="docs-toc-nav" aria-label="On-page anchors">
                {headings.map((h, i) => (
                  <a
                    key={`${h.id}-${i}`}
                    href={`#${h.id}`}
                    className={`docs-toc-link docs-toc-${h.level} ${activeId === h.id ? 'active' : ''}`}
                    onClick={(e) => scrollToHeading(h.id, e)}
                  >
                    {h.text}
                  </a>
                ))}
              </nav>
            ) : (
              <p className="docs-toc-empty">Overview</p>
            )}

            <div className="docs-toc-footer">
              <a
                href={githubUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="docs-toc-edit-link"
              >
                <svg
                  width="13"
                  height="13"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  aria-hidden="true"
                >
                  <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
                  <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
                </svg>
                <span>Edit on GitHub</span>
              </a>
            </div>
          </div>
        </aside>
      </div>
      <Footer />
    </div>
  )
}
