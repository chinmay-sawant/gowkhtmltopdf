import { useState, useCallback, useEffect, useRef } from 'react'
import { SEVERITY_META, STATUS_META } from '../data/constants'
import RichText from './RichText'
import { highlightText } from './highlightText'

function SeverityBadge({ severity }) {
  const meta = SEVERITY_META[severity]
  if (!meta) return null
  return (
    <span className="badge" style={{ background: meta.bg, color: meta.text }}>
      {severity}
    </span>
  )
}

function StatusBadge({ status }) {
  const meta = STATUS_META[status] ?? { label: status, accent: 'var(--surface-2)', text: 'var(--muted)' }
  return (
    <span className="status-badge" style={{ background: meta.accent, color: meta.text }}>
      {meta.label}
    </span>
  )
}

export default function IssueCard({ issue, query = '', isTarget = false }) {
  const [copied, setCopied] = useState(false)
  const cardRef = useRef(null)

  useEffect(() => {
    if (isTarget && cardRef.current) {
      cardRef.current.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }, [isTarget])

  const handleCopyLink = useCallback(
    (e) => {
      e.preventDefault()
      e.stopPropagation()
      const url = `${window.location.origin}${window.location.pathname}#/dossier?issue=${issue.number}`
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard
          .writeText(url)
          .then(() => {
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
          })
          .catch(() => {})
      } else {
        const ta = document.createElement('textarea')
        ta.value = url
        ta.style.position = 'fixed'
        ta.style.opacity = '0'
        document.body.appendChild(ta)
        ta.focus()
        ta.select()
        try {
          document.execCommand('copy')
          setCopied(true)
          setTimeout(() => setCopied(false), 2000)
        } catch {
          // Ignore
        }
        document.body.removeChild(ta)
      }
    },
    [issue.number],
  )

  return (
    <article
      ref={cardRef}
      id={`issue-${issue.number}`}
      className={`issue${isTarget ? ' is-target-issue' : ''}`}
      data-status={issue.status}
    >
      <h3 className="title">
        {highlightText(issue.title, query, `t-${issue.number}`)}
      </h3>
      <div className="meta">
        <div className="issue-num-wrapper">
          <a
            className="num"
            href={`https://github.com/wkhtmltopdf/wkhtmltopdf/issues/${issue.number}`}
            target="_blank"
            rel="noopener noreferrer"
            title={`View issue #${issue.number} on GitHub`}
          >
            #{issue.number}
          </a>
          <button
            type="button"
            className={`issue-copy-link-btn${copied ? ' copied' : ''}`}
            onClick={handleCopyLink}
            title={copied ? 'Link copied!' : 'Copy direct link to issue'}
            aria-label={`Copy direct link to issue #${issue.number}`}
          >
            {copied ? (
              <>
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="20 6 9 17 4 12" />
                </svg>
                <span className="copy-tooltip" role="status" aria-live="polite">Copied!</span>
              </>
            ) : (
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
                <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
              </svg>
            )}
          </button>
        </div>
        <span className="cat-tag">{issue.category}</span>
        <SeverityBadge severity={issue.severity} />
        <StatusBadge status={issue.status} />
      </div>
      <p className="summary">
        <RichText highlight={query}>{issue.summary}</RichText>
      </p>
      {issue.evidence && (
        <p className="ev">
          <span className="ev-label">Codebase status</span>
          <RichText highlight={query}>{issue.evidence}</RichText>
        </p>
      )}
      {issue.workaround && (
        <p className="wa">
          <span className="wa-label">Workaround</span>
          <RichText highlight={query}>{issue.workaround}</RichText>
        </p>
      )}
      {issue.key_detail && (
        <p className="kd">
          <span className="kd-label">Detail</span>
          <RichText highlight={query}>{issue.key_detail}</RichText>
        </p>
      )}
      {issue.author && (
        <p className="kd">
          <span className="kd-label">Author</span>
          {issue.author}
          {issue.created_at ? ` / ${new Date(issue.created_at).toISOString().slice(0, 10)}` : ''}
          {issue.comments ? ` / ${issue.comments} comments` : ''}
        </p>
      )}
    </article>
  )
}

