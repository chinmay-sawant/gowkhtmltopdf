export default function Pagination({ page, pageCount, onPageChange, pageSize }) {
  if (pageCount <= 1) return null

  const window = 2

  function pages() {
    const out = new Set([1, pageCount])
    for (let p = page - window; p <= page + window; p++) {
      if (p >= 1 && p <= pageCount) out.add(p)
    }
    return [...out].sort((a, b) => a - b)
  }

  const list = pages()

  return (
    <nav className="pagination" aria-label="Pagination">
      <button
        type="button"
        className="page-btn"
        disabled={page === 1}
        onClick={() => onPageChange(page - 1)}
      >
        Prev
      </button>
      {list.map((p, i) => {
        const hasGap = i > 0 && p - list[i - 1] > 1
        return (
          <span className="page-group" key={p}>
            {hasGap && <span className="page-ellipsis">…</span>}
            <button
              type="button"
              className={p === page ? 'page-btn active' : 'page-btn'}
              onClick={() => onPageChange(p)}
              aria-current={p === page ? 'page' : undefined}
            >
              {p}
            </button>
          </span>
        )
      })}
      <button
        type="button"
        className="page-btn"
        disabled={page === pageCount}
        onClick={() => onPageChange(page + 1)}
      >
        Next
      </button>
      <span className="page-info">
        page {page} of {pageCount}
        <span className="page-info-sep">/</span>~{pageSize} per page
      </span>
    </nav>
  )
}
