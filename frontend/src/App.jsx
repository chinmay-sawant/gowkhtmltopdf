import { lazy, Suspense, useEffect } from 'react'
import { HashRouter, Routes, Route, Navigate, Outlet, useLocation } from 'react-router-dom'
import SiteNav from './components/SiteNav'
import Footer from './components/Footer'
import CommandPalette from './components/CommandPalette'
import ContentPage from './pages/ContentPage'
import LandingPage from './pages/LandingPage'
import NotFoundPage from './pages/NotFoundPage'

const DocumentationPage = lazy(() => import('./pages/DocumentationPage'))
const DossierPage = lazy(() => import('./pages/DossierPage'))
const ShowcasePage = lazy(() => import('./pages/ShowcasePage'))
const BenchmarksPage = lazy(() => import('./pages/BenchmarksPage'))

const DOC_REDIRECTS = [
  ['cli', 'cli'],
  ['library-api', 'library-api'],
  ['architecture', 'architecture'],
  ['compatibility', 'compatibility'],
  ['fonts', 'fonts'],
  ['security', 'security'],
  ['performance', 'performance'],
]

function ScrollToTop() {
  const { pathname } = useLocation()
  useEffect(() => {
    window.scrollTo({ top: 0, left: 0, behavior: 'instant' })
  }, [pathname])
  return null
}

function WrapLayout() {
  return (
    <div className="wrap">
      <main id="main-content" tabIndex={-1}>
        <Outlet />
      </main>
      <Footer />
    </div>
  )
}

export default function App() {
  return (
    <HashRouter>
      <ScrollToTop />
      <CommandPalette />
      <button
        type="button"
        className="skip-link"
        onClick={() => {
          const target = document.getElementById('main-content')
          if (target) {
            target.tabIndex = -1
            target.focus()
            target.scrollIntoView({ behavior: 'smooth' })
          }
        }}
      >
        Skip to content
      </button>
      <SiteNav />

      <Suspense fallback={<div className="wrap">Loading…</div>}>
        <Routes>
          <Route element={<WrapLayout />}>
            <Route path="/" element={<LandingPage />} />
            <Route path="/getting-started" element={<ContentPage />} />
            <Route path="/about" element={<ContentPage />} />
            <Route path="/dossier" element={<DossierPage />} />
            <Route path="/showcase" element={<ShowcasePage />} />
            <Route path="/benchmarks" element={<BenchmarksPage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Route>
          <Route path="/documentation" element={<Navigate to="/documentation/cli" replace />} />
          <Route path="/documentation/:docId" element={<DocumentationPage />} />
          {DOC_REDIRECTS.map(([from, to]) => (
            <Route
              key={from}
              path={`/${from}`}
              element={<Navigate to={`/documentation/${to}`} replace />}
            />
          ))}
        </Routes>
      </Suspense>
    </HashRouter>
  )
}

