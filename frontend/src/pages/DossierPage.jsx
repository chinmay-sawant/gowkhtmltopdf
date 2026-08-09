import { useMemo, useState, useEffect } from 'react'
import ContentBlocks from '../components/blocks/ContentBlocks'
import FilterChips from '../components/FilterChips'
import StatsSidebar from '../components/StatsSidebar'
import IssueCard from '../components/IssueCard'
import Pagination from '../components/Pagination'
import PageTitle from '../components/PageTitle'
import { sortIssues, countBy } from '../data/issues'
import { CATEGORY_ORDER, STATUS_META } from '../data/constants'
import dossier from '../data/content/page-dossier.json'

const PAGE_SIZES = [10, 25, 50, 100]

export default function DossierPage() {
  const issues = useMemo(() => sortIssues(), [])
  const [status, setStatus] = useState('all')
  const [category, setCategory] = useState('all')
  const [severity, setSeverity] = useState('all')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(25)

  const filtered = useMemo(
    () =>
      issues.filter(
        (it) =>
          (status === 'all' || it.status === status) &&
          (category === 'all' || it.category === category) &&
          (severity === 'all' || it.severity === severity),
      ),
    [issues, status, category, severity],
  )

  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize))
  const safePage = Math.min(page, pageCount)
  const pageItems = useMemo(
    () => filtered.slice((safePage - 1) * pageSize, safePage * pageSize),
    [filtered, safePage, pageSize],
  )

  useEffect(() => {
    setPage(1)
  }, [status, category, severity, pageSize])

  const catCounts = countBy(issues, 'category')
  const statusCounts = countBy(issues, 'status')
  const sevCounts = countBy(issues, 'severity')

  const statusSave = useMemo(() => {
    const map = {}
    for (const [k, v] of Object.entries(STATUS_META)) map[k] = v.label
    return map
  }, [])

  const selectionNote =
    status === 'all' && category === 'all' && severity === 'all'
      ? `Showing all ${issues.length} issues. Filter by coverage status, area, or severity, or page through the results.`
      : `Showing ${filtered.length} of ${issues.length} issue${filtered.length === 1 ? '' : 's'}${[
          status !== 'all' ? statusSave[status].toLowerCase() : '',
          category !== 'all' ? category.toLowerCase() : '',
          severity !== 'all' ? `${severity.toLowerCase()} severity` : '',
        ]
          .filter(Boolean)
          .join(' + ')}`

  return (
    <>
      <PageTitle title="Issue Dossier" />
      <div className="page-block intro">
        <ContentBlocks content={dossier.content} />
      </div>

      <FilterChips
        status={status}
        statusCounts={statusCounts}
        onStatusChange={setStatus}
        category={category}
        categories={CATEGORY_ORDER.filter((c) => catCounts[c])}
        catCounts={catCounts}
        onCategoryChange={setCategory}
      />

      <p className="controls-note">{selectionNote}</p>

      <div className="pager-top">
        <Pagination page={safePage} pageCount={pageCount} onPageChange={setPage} pageSize={pageSize} />
        <label className="page-size">
          Per page
          <select value={pageSize} onChange={(e) => setPageSize(Number(e.target.value))}>
            {PAGE_SIZES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="layout">
        <div className="issues">
          {pageItems.map((it) => (
            <IssueCard key={it.number} issue={it} />
          ))}
        </div>
        <StatsSidebar
          statusCounts={statusCounts}
          catCounts={catCounts}
          sevCounts={sevCounts}
          total={issues.length}
          status={status}
          category={category}
          severity={severity}
          onStatusChange={setStatus}
          onCategoryChange={setCategory}
          onSeverityChange={setSeverity}
        />
      </div>

      <Pagination page={safePage} pageCount={pageCount} onPageChange={setPage} pageSize={pageSize} />
    </>
  )
}
