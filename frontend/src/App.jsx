import { HashRouter, Routes, Route, Navigate } from 'react-router-dom'
import SiteNav from './components/SiteNav'
import Footer from './components/Footer'
import ContentPage from './pages/ContentPage'
import DossierPage from './pages/DossierPage'
import ShowcasePage from './pages/ShowcasePage'

export default function App() {
  return (
    <HashRouter>
      <SiteNav />
      <div className="wrap">
        <Routes>
          <Route path="/" element={<ContentPage />} />
          <Route path="/getting-started" element={<ContentPage />} />
          <Route path="/cli" element={<ContentPage />} />
          <Route path="/library-api" element={<ContentPage />} />
          <Route path="/architecture" element={<ContentPage />} />
          <Route path="/compatibility" element={<ContentPage />} />
          <Route path="/fonts" element={<ContentPage />} />
          <Route path="/security" element={<ContentPage />} />
          <Route path="/about" element={<ContentPage />} />
          <Route path="/dossier" element={<DossierPage />} />
          <Route path="/showcase" element={<ShowcasePage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
        <Footer />
      </div>
    </HashRouter>
  )
}
