import { Link } from 'react-router-dom'
import PageTitle from '../components/PageTitle'

export default function NotFoundPage() {
  return (
    <div className="not-found-page hero">
      <PageTitle title="Page Not Found" />
      <h1>404 — Page not found</h1>
      <p className="lede">The page you are looking for does not exist or has been moved.</p>
      <div className="landing-actions" style={{ marginTop: '24px' }}>
        <Link className="button button-primary" to="/">
          Back to Overview <span aria-hidden="true">→</span>
        </Link>
        <Link className="button button-secondary" to="/getting-started">
          Getting Started
        </Link>
      </div>
    </div>
  )
}
