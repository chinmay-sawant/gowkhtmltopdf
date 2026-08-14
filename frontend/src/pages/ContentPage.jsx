import { useMemo } from 'react'
import { useLocation } from 'react-router-dom'
import ContentBlocks from '../components/blocks/ContentBlocks'
import PageTitle from '../components/PageTitle'
import NotFoundPage from './NotFoundPage'

function loadContent(id) {
  const pages = import.meta.glob('../data/content/page-*.json', { eager: true })
  for (const path in pages) {
    if (pages[path].id === id) return pages[path]
  }
  return null
}

const ALIAS = { '/': 'overview' }

export default function ContentPage() {
  const { pathname } = useLocation()
  const id = ALIAS[pathname] ?? pathname.replace(/^\//, '')
  const page = useMemo(() => loadContent(id), [id])

  if (!page) return <NotFoundPage />
  return (
    <div className="content-page">
      <PageTitle title={page.nav} />
      <ContentBlocks content={page.content} />
    </div>
  )
}

