import { useMemo } from 'react'
import { useParams, NavLink, Navigate } from 'react-router-dom'
import ContentBlocks from '../components/blocks/ContentBlocks'
import PageTitle from '../components/PageTitle'
import Footer from '../components/Footer'

function loadContent(id) {
  const pages = import.meta.glob('../data/content/page-*.json', { eager: true })
  for (const path in pages) {
    if (pages[path].id === id) return pages[path]
  }
  return null
}

const DOCS = ['cli', 'library-api', 'architecture', 'compatibility', 'fonts', 'security']

export default function DocumentationPage() {
  const { docId } = useParams()
  const docs = useMemo(
    () =>
      DOCS.map((id) => {
        const page = loadContent(id)
        return { id, nav: page ? page.nav : id }
      }),
    [],
  )

  const page = useMemo(() => loadContent(docId), [docId])
  if (!page) return <Navigate to="/documentation/cli" replace />

  return (
    <div className="docs-page" id="main-content">
      <PageTitle title={page.nav} />
      <div className="docs-layout">
        <aside className="docs-sidebar">
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
        <div className="docs-main">
          <div className="docs-context" aria-label="Current documentation section">
            <span>Documentation</span>
            <span aria-hidden="true">/</span>
            <strong>{page.nav}</strong>
          </div>
          <ContentBlocks content={page.content} />
        </div>
      </div>
      <Footer />
    </div>
  )
}
