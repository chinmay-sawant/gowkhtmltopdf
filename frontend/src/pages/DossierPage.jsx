import { useMemo, useCallback } from 'react'
import { useSearchParams } from 'react-router-dom'
import ContentBlocks from '../components/blocks/ContentBlocks'
import FilterChips from '../components/FilterChips'
import StatsSidebar from '../components/StatsSidebar'
import IssueCard from '../components/IssueCard'
import Pagination from '../components/Pagination'
import PageTitle from '../components/PageTitle'
import { countBy } from '../data/issues'
import { useIssues } from '../hooks/useIssues'
import { CATEGORY_ORDER, STATUS_META } from '../data/constants'
import dossier from '../data/content/page-dossier.json'

const PAGE_SIZES = [10, 25, 50, 100]

export default function DossierPage() {
  const { issues, loading, error, reload } = useIssues()
  const [searchParams, setSearchParams] = useSearchParams()

  const status = searchParams.get('status') || 'all'
  const category = searchParams.get('category') || 'all'
  const severity = searchParams.get('severity') || 'all'
  const q = searchParams.get('q') || ''
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

    return issues.filter((it) => {
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
  }, [issues, status, category, severity, q])

  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize))
  const safePage = Math.min(page, pageCount)
  const pageItems = useMemo(
    () => filtered.slice((safePage - 1) * pageSize, safePage * pageSize),
    [filtered, safePage, pageSize],
  )

  const catCounts = useMemo(() => countBy(issues, 'category'), [issues])
  const statusCounts = useMemo(() => countBy(issues, 'status'), [issues])
  const sevCounts = useMemo(() => countBy(issues, 'severity'), [issues])

  const statusSave = useMemo(() => {
    const map = {}
    for (const [k, v] of Object.entries(STATUS_META)) map[k] = v.label
    return map
  }, [])

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
                  <IssueCard key={it.number} issue={it} />
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
