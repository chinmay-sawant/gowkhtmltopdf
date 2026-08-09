import { SEVERITY_META, STATUS_META } from '../data/constants'

function SeverityBadge({ severity }) {
  const meta = SEVERITY_META[severity]
  return (
    <span className="badge" style={{ background: meta.bg, color: meta.text }}>
      {severity}
    </span>
  )
}

function StatusBadge({ status }) {
  const meta = STATUS_META[status] ?? { label: status, accent: '#F0EFEA', text: '#787774' }
  return (
    <span className="status-badge" style={{ background: meta.accent, color: meta.text }}>
      {meta.label}
    </span>
  )
}

export default function IssueCard({ issue }) {
  return (
    <article className="issue" data-status={issue.status}>
      <h3 className="title">{issue.title}</h3>
      <div className="meta">
        <a
          className="num"
          href={`https://github.com/wkhtmltopdf/wkhtmltopdf/issues/${issue.number}`}
          target="_blank"
          rel="noopener noreferrer"
        >
          #{issue.number}
        </a>
        <span className="cat-tag">{issue.category}</span>
        <SeverityBadge severity={issue.severity} />
        <StatusBadge status={issue.status} />
      </div>
      <p className="summary">{issue.summary}</p>
      {issue.evidence && (
        <p className="ev">
          <span className="ev-label">Codebase status</span>
          {issue.evidence}
        </p>
      )}
      {issue.workaround && (
        <p className="wa">
          <span className="wa-label">Workaround</span>
          {issue.workaround}
        </p>
      )}
      {issue.key_detail && (
        <p className="kd">
          <span className="kd-label">Detail</span>
          <code>{issue.key_detail}</code>
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
