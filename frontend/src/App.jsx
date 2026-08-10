import { HashRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom'
import SiteNav from './components/SiteNav'
import Footer from './components/Footer'
import ContentPage from './pages/ContentPage'
import DocumentationPage from './pages/DocumentationPage'
import DossierPage from './pages/DossierPage'
import ShowcasePage from './pages/ShowcasePage'
import LandingPage from './pages/LandingPage'

const DOC_REDIRECTS = [
  ['cli', 'cli'],
  ['library-api', 'library-api'],
  ['architecture', 'architecture'],
  ['compatibility', 'compatibility'],
  ['fonts', 'fonts'],
  ['security', 'security'],
]

function WrapLayout() {
  return (
    <div className="wrap">
      <main id="main-content">
        <Outlet />
      </main>
      <Footer />
    </div>
  )
}

export default function App() {
  return (
    <HashRouter>
      <a className="skip-link" href="#main-content">Skip to content</a>
      <SiteNav />
      <Routes>
        <Route element={<WrapLayout />}>
          <Route path="/" element={<LandingPage />} />
          <Route path="/getting-started" element={<ContentPage />} />
          <Route path="/about" element={<ContentPage />} />
          <Route path="/dossier" element={<DossierPage />} />
          <Route path="/showcase" element={<ShowcasePage />} />
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
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </HashRouter>
  )
}
