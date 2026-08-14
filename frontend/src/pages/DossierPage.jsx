import { useMemo, useCallback, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import ContentBlocks from '../components/blocks/ContentBlocks'
import FilterChips from '../components/FilterChips'
import StatsSidebar from '../components/StatsSidebar'
import IssueCard from '../components/IssueCard'
import Pagination from '../components/Pagination'
import PageTitle from '../components/PageTitle'
import { countBy } from '../data/issues'
import { useIssues } from '../hooks/useIssues'
import { CATEGORY_ORDER, STATUS_META, STATUS_ORDER } from '../data/constants'
import dossier from '../data/content/page-dossier.json'

const PAGE_SIZES = [10, 25, 50, 100]
const SEVERITY_WEIGHT = { High: 3, Medium: 2, Low: 1 }

export default function DossierPage() {
  const { issues, loading, error, reload } = useIssues()
  const [searchParams, setSearchParams] = useSearchParams()

  const status = searchParams.get('status') || 'all'
  const category = searchParams.get('category') || 'all'
  const severity = searchParams.get('severity') || 'all'
  const sort = searchParams.get('sort') || 'number-desc'
  const q = searchParams.get('q') || ''
  const targetIssueParam = searchParams.get('issue')
  const targetIssueNum = targetIssueParam ? parseInt(targetIssueParam, 10) : null
  const rawPage = parseInt(searchParams.get('page') || '1', 10)
  const page = Number.isFinite(rawPage) && rawPage > 0 ? rawPage : 1
  const rawPageSize = parseInt(searchParams.get('pageSize') || '25', 10)
  const pageSize = PAGE_SIZES.includes(rawPageSize) ? rawPageSize : 25

  const updateFilters = useCallback(
    (updates) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          for (const [k, v] of Object.entries(updates)) {
            if (
              v === undefined ||
              v === null ||
              v === '' ||
              (k === 'status' && v === 'all') ||
              (k === 'category' && v === 'all') ||
              (k === 'severity' && v === 'all') ||
              (k === 'sort' && v === 'number-desc') ||
              (k === 'page' && Number(v) === 1) ||
              (k === 'pageSize' && Number(v) === 25)
            ) {
              next.delete(k)
            } else {
              next.set(k, String(v))
            }
          }
          return next
        },
        { replace: false },
      )
    },
    [setSearchParams],
  )

  const handleStatusChange = useCallback(
    (newStatus) => {
      updateFilters({ status: newStatus, page: 1 })
    },
    [updateFilters],
  )

  const handleCategoryChange = useCallback(
    (newCategory) => {
      updateFilters({ category: newCategory, page: 1 })
    },
    [updateFilters],
  )

  const handleSeverityChange = useCallback(
    (newSeverity) => {
      updateFilters({ severity: newSeverity, page: 1 })
    },
    [updateFilters],
  )

  const handleSortChange = useCallback(
    (newSort) => {
      updateFilters({ sort: newSort, page: 1 })
    },
    [updateFilters],
  )

  const handleSearchChange = useCallback(
    (e) => {
      const newQ = e.target.value
      updateFilters({ q: newQ, page: 1 })
    },
    [updateFilters],
  )

  const handleSearchClear = useCallback(() => {
    updateFilters({ q: '', page: 1 })
  }, [updateFilters])

  const handlePageChange = useCallback(
    (newPage) => {
      updateFilters({ page: newPage })
    },
    [updateFilters],
  )

  const handlePageSizeChange = useCallback(
    (newPageSize) => {
      updateFilters({ pageSize: newPageSize, page: 1 })
    },
    [updateFilters],
  )

  const handleClearAll = useCallback(() => {
    setSearchParams({}, { replace: false })
  }, [setSearchParams])

  const filtered = useMemo(() => {
    const qLower = q.trim().toLowerCase()
    const cleanQ = qLower.startsWith('#') ? qLower.slice(1) : qLower

    const result = issues.filter((it) => {
      if (status !== 'all' && it.status !== status) return false
      if (category !== 'all' && it.category !== category) return false
      if (severity !== 'all' && it.severity !== severity) return false

      if (!qLower) return true

      const numStr = it.number != null ? String(it.number) : ''
      const title = it.title ? it.title.toLowerCase() : ''
      const summary = it.summary ? it.summary.toLowerCase() : ''
      const evidence = it.evidence ? it.evidence.toLowerCase() : ''
      const workaround = it.workaround ? it.workaround.toLowerCase() : ''
      const keyDetail = it.key_detail ? it.key_detail.toLowerCase() : ''

      return (
        title.includes(qLower) ||
        (cleanQ && numStr.includes(cleanQ)) ||
        summary.includes(qLower) ||
        evidence.includes(qLower) ||
        workaround.includes(qLower) ||
        keyDetail.includes(qLower)
      )
    })

    return result.sort((a, b) => {
      if (sort === 'number-asc') {
        return a.number - b.number
      }
      if (sort === 'severity-desc') {
        const diff = (SEVERITY_WEIGHT[b.severity] || 0) - (SEVERITY_WEIGHT[a.severity] || 0)
        if (diff !== 0) return diff
        return b.number - a.number
      }
      if (sort === 'comments-desc') {
        const diff = (b.comments || 0) - (a.comments || 0)
        if (diff !== 0) return diff
        return b.number - a.number
      }
      // Default: number-desc
      return b.number - a.number
    })
  }, [issues, status, category, severity, q, sort])

  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize))
  const safePage = Math.min(page, pageCount)
  const pageItems = useMemo(
    () => filtered.slice((safePage - 1) * pageSize, safePage * pageSize),
    [filtered, safePage, pageSize],
  )

  const catCounts = useMemo(() => countBy(issues, 'category'), [issues])
  const statusCounts = useMemo(() => countBy(issues, 'status'), [issues])
  const sevCounts = useMemo(() => countBy(issues, 'severity'), [issues])

  // Automatically scroll / paginate to target issue if deep-linked via URL
  useEffect(() => {
    if (targetIssueNum && !loading && filtered.length > 0) {
      const targetIdx = filtered.findIndex((it) => it.number === targetIssueNum)
      if (targetIdx >= 0) {
        const targetPage = Math.floor(targetIdx / pageSize) + 1
        if (targetPage !== safePage) {
          updateFilters({ page: targetPage })
        }
      }
    }
  }, [targetIssueNum, loading, filtered, pageSize, safePage, updateFilters])

  const statusSave = useMemo(() => {
    const map = {}
    for (const [k, v] of Object.entries(STATUS_META)) map[k] = v.label
    return map
  }, [])

  const totalIssues = issues.length
  const implCount = statusCounts['implemented'] || 0
  const partCount = statusCounts['partial'] || 0
  const notImplCount = statusCounts['not-implemented'] || 0

  const implPct = totalIssues > 0 ? ((implCount / totalIssues) * 100).toFixed(1) : '0.0'
  const partPct = totalIssues > 0 ? ((partCount / totalIssues) * 100).toFixed(1) : '0.0'
  const notImplPct = totalIssues > 0 ? ((notImplCount / totalIssues) * 100).toFixed(1) : '0.0'

  const selectionNote = useMemo(() => {
    if (loading) return 'Loading issue dataset…'
    if (error) return 'Failed to load issues'
    if (status === 'all' && category === 'all' && severity === 'all' && !q) {
      return `Showing all ${issues.length} issues. Filter by coverage status, area, or severity, search keywords, or page through the results.`
    }
    const filtersApplied = [
      status !== 'all' ? statusSave[status]?.toLowerCase() : '',
      category !== 'all' ? category.toLowerCase() : '',
      severity !== 'all' ? `${severity.toLowerCase()} severity` : '',
      q ? `matching "${q}"` : '',
    ]
      .filter(Boolean)
      .join(' + ')

    return `Showing ${filtered.length} of ${issues.length} issue${filtered.length === 1 ? '' : 's'}${filtersApplied ? ` (${filtersApplied})` : ''}`
  }, [loading, error, status, category, severity, q, issues.length, filtered.length, statusSave])

  const methodologyBlocks = useMemo(
    () => dossier.content.filter((b) => b.type !== 'hero' && b.type !== 'callout'),
    [],
  )

  return (
    <div className="dossier-page">
      <PageTitle title="Issue Dossier" />

      <header className="dossier-header">
        <div className="dossier-hero">
          <h1>Issue Dossier</h1>
          <p className="lede">
            All 1,329 open wkhtmltopdf issues classified against gowkhtmltopdf coverage with cited code evidence.
          </p>
        </div>

        <aside className="dossier-banner" role="note">
          <span className="dossier-banner-pill">AI Verdict Notice</span>
          <span className="dossier-banner-text">
            Statuses and evidence were assigned by AI subagents from issue metadata and codebase scans. Verify cited code paths before relying on them.
          </span>
        </aside>

        {/* FE-25: Interactive Coverage Distribution Bar */}
        <section className="coverage-bar-card" aria-label="gowkhtmltopdf Coverage Distribution">
          <div className="coverage-bar-meta">
            <div className="coverage-bar-title-group">
              <span className="coverage-bar-label">Coverage Distribution</span>
              <span className="coverage-bar-sub">Interactive status breakdown — click any segment to filter</span>
            </div>
            <div className="coverage-bar-stats-summary">
              <button
                type="button"
                className={`coverage-stat-pill segment-implemented${status === 'implemented' ? ' active' : ''}`}
                onClick={() => handleStatusChange(status === 'implemented' ? 'all' : 'implemented')}
                title="Filter by Implemented"
              >
                <span className="dot" style={{ background: '#9BBF88' }} />
                <span>Implemented:</span>
                <strong>{implPct}%</strong>
                <span className="stat-pill-count">({implCount})</span>
              </button>
              <button
                type="button"
                className={`coverage-stat-pill segment-partial${status === 'partial' ? ' active' : ''}`}
                onClick={() => handleStatusChange(status === 'partial' ? 'all' : 'partial')}
                title="Filter by Partial"
              >
                <span className="dot" style={{ background: '#E7CD80' }} />
                <span>Partial:</span>
                <strong>{partPct}%</strong>
                <span className="stat-pill-count">({partCount})</span>
              </button>
              <button
                type="button"
                className={`coverage-stat-pill segment-not-implemented${status === 'not-implemented' ? ' active' : ''}`}
                onClick={() => handleStatusChange(status === 'not-implemented' ? 'all' : 'not-implemented')}
                title="Filter by Not Implemented"
              >
                <span className="dot" style={{ background: '#D89A8B' }} />
                <span>Not Impl.:</span>
                <strong>{notImplPct}%</strong>
                <span className="stat-pill-count">({notImplCount})</span>
              </button>
            </div>
          </div>
          <div className="coverage-segmented-bar" role="group" aria-label="Filter by coverage status">
            <button
              type="button"
              className={`coverage-segment segment-implemented${status === 'implemented' ? ' active' : ''}`}
              style={{ width: `${implPct}%` }}
              onClick={() => handleStatusChange(status === 'implemented' ? 'all' : 'implemented')}
              aria-label={`Implemented: ${implCount} issues (${implPct}%). Click to ${status === 'implemented' ? 'clear filter' : 'filter'}`}
              aria-pressed={status === 'implemented'}
              title={`Implemented: ${implCount} issues (${implPct}%)`}
            >
              <span className="segment-text">{implPct}% Implemented</span>
            </button>
            <button
              type="button"
              className={`coverage-segment segment-partial${status === 'partial' ? ' active' : ''}`}
              style={{ width: `${partPct}%` }}
              onClick={() => handleStatusChange(status === 'partial' ? 'all' : 'partial')}
              aria-label={`Partial: ${partCount} issues (${partPct}%). Click to ${status === 'partial' ? 'clear filter' : 'filter'}`}
              aria-pressed={status === 'partial'}
              title={`Partial: ${partCount} issues (${partPct}%)`}
            >
              <span className="segment-text">{partPct}% Partial</span>
            </button>
            <button
              type="button"
              className={`coverage-segment segment-not-implemented${status === 'not-implemented' ? ' active' : ''}`}
              style={{ width: `${notImplPct}%` }}
              onClick={() => handleStatusChange(status === 'not-implemented' ? 'all' : 'not-implemented')}
              aria-label={`Not implemented: ${notImplCount} issues (${notImplPct}%). Click to ${status === 'not-implemented' ? 'clear filter' : 'filter'}`}
              aria-pressed={status === 'not-implemented'}
              title={`Not implemented: ${notImplCount} issues (${notImplPct}%)`}
            >
              <span className="segment-text">{notImplPct}% Not Impl.</span>
            </button>
          </div>
        </section>
      </header>

      <section className="dossier-controls" aria-label="Issue filters and search">
        <div className="dossier-search-bar">
          <div className="dossier-search-wrapper">
            <span className="dossier-search-icon" aria-hidden="true">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <circle cx="11" cy="11" r="8" />
                <line x1="21" y1="21" x2="16.65" y2="16.65" />
              </svg>
            </span>
            <input
              id="dossier-search-input"
              type="search"
              placeholder="Search title, #number, summary, evidence..."
              value={q}
              onChange={handleSearchChange}
              autoComplete="off"
              spellCheck={false}
              aria-label="Search issues"
              className="dossier-search-input"
            />
            {q && (
              <button
                type="button"
                className="dossier-search-clear"
                onClick={handleSearchClear}
                aria-label="Clear search"
                title="Clear search"
              >
                ✕
              </button>
            )}
          </div>
        </div>

        <FilterChips
          status={status}
          statusCounts={statusCounts}
          onStatusChange={handleStatusChange}
          category={category}
          categories={CATEGORY_ORDER.filter((c) => catCounts[c])}
          catCounts={catCounts}
          onCategoryChange={handleCategoryChange}
        />
      </section>

      <div className="dossier-toolbar">
        <p className="controls-note">{selectionNote}</p>
        <div className="toolbar-actions">
          <label className="sort-control">
            <span className="sort-label">Sort</span>
            <select
              value={sort}
              onChange={(e) => handleSortChange(e.target.value)}
              className="sort-select"
              aria-label="Sort issues by"
            >
              <option value="number-desc">Newest issue (#5283 → #1)</option>
              <option value="number-asc">Oldest issue (#1 → #5283)</option>
              <option value="severity-desc">Highest severity (High → Low)</option>
              <option value="comments-desc">Most discussed (Comments)</option>
            </select>
          </label>
          <div className="pager-top">
            <Pagination page={safePage} pageCount={pageCount} onPageChange={handlePageChange} pageSize={pageSize} />
            <label className="page-size">
              Per page
              <select value={pageSize} onChange={(e) => handlePageSizeChange(Number(e.target.value))}>
                {PAGE_SIZES.map((s) => (
                  <option key={s} value={s}>
                    {s}
                  </option>
                ))}
              </select>
            </label>
          </div>
        </div>
      </div>

      <div className="layout">
        <main className="dossier-main-content">
          {loading ? (
            <div className="dossier-loading" aria-live="polite" aria-busy="true">
              <div className="dossier-spinner" aria-hidden="true" />
              <p>Loading issue dossier (1,329 classified records)…</p>
            </div>
          ) : error ? (
            <div className="dossier-error" role="alert">
              <h3>Unable to load issue dossier</h3>
              <p>{error}</p>
              <button type="button" className="dossier-retry-btn" onClick={reload}>
                Retry
              </button>
            </div>
          ) : filtered.length === 0 ? (
            <div className="dossier-empty-state" role="status">
              <div className="dossier-empty-icon" aria-hidden="true">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
                  <circle cx="11" cy="11" r="8" />
                  <line x1="21" y1="21" x2="16.65" y2="16.65" />
                  <line x1="8" y1="11" x2="14" y2="11" />
                </svg>
              </div>
              <h3>No issues match your current filters.</h3>
              {q && <p className="dossier-empty-query">No records found matching &ldquo;{q}&rdquo;</p>}
              <button
                type="button"
                className="dossier-clear-btn"
                onClick={handleClearAll}
              >
                Clear all filters
              </button>
            </div>
          ) : (
            <>
              <div className="issues">
                {pageItems.map((it) => (
                  <IssueCard
                    key={it.number}
                    issue={it}
                    query={q}
                    isTarget={targetIssueNum === it.number}
                  />
                ))}
              </div>
              <Pagination page={safePage} pageCount={pageCount} onPageChange={handlePageChange} pageSize={pageSize} />
            </>
          )}
        </main>

        <StatsSidebar
          statusCounts={statusCounts}
          catCounts={catCounts}
          sevCounts={sevCounts}
          total={issues.length}
          status={status}
          category={category}
          severity={severity}
          onStatusChange={handleStatusChange}
          onCategoryChange={handleCategoryChange}
          onSeverityChange={handleSeverityChange}
        />
      </div>

      <details className="dossier-methodology">
        <summary>Classification methodology & background</summary>
        <div className="dossier-methodology-content">
          <ContentBlocks content={methodologyBlocks} />
        </div>
      </details>
    </div>
  )
}

