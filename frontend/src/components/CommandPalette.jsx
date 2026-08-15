import { useState, useEffect, useRef, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { SHOWCASE, SHOWCASE_SPECIAL } from '../data/showcase'
import useTheme from '../hooks/useTheme'

const DOC_ITEMS = [
  {
    id: 'doc-cli',
    title: 'CLI Reference',
    desc: 'Multi-object command grammar, global vs page-scoped flags, cover & TOC objects',
    category: 'Documentation',
    url: '/documentation/cli',
    keywords: 'cli options flags terminal usage grammar objects page toc cover arguments',
    icon: 'cli',
  },
  {
    id: 'doc-library-api',
    title: 'Go Library API',
    desc: 'In-process Go conversion, typed settings, context cancellation, streaming',
    category: 'Documentation',
    url: '/documentation/library-api',
    keywords: 'go library api sdk package settings context cancellation in-process inproc embed',
    icon: 'api',
  },
  {
    id: 'doc-architecture',
    title: 'Architecture & Pipeline',
    desc: '10 domain packages, pipeline strip, layout domains, dependency DAG',
    category: 'Documentation',
    url: '/documentation/architecture',
    keywords: 'architecture pipeline domains layout dag parsing paint render engine design',
    icon: 'arch',
  },
  {
    id: 'doc-compatibility',
    title: 'HTML & CSS Compatibility',
    desc: 'Print pagination, page-breaks, supported CSS features, fidelity comparison',
    category: 'Documentation',
    url: '/documentation/compatibility',
    keywords: 'compatibility html css flexbox tables page breaks print fidelity support',
    icon: 'compat',
  },
  {
    id: 'doc-fonts',
    title: 'Fonts & Typography',
    desc: 'Font loader, OpenType/TrueType parsing, font matching, Google Fonts sampler',
    category: 'Documentation',
    url: '/documentation/fonts',
    keywords: 'fonts typography opentype truetype google fonts ttf otf text glyphs',
    icon: 'font',
  },
  {
    id: 'doc-security',
    title: 'Security Model',
    desc: 'Threat model, local file access sandboxing, SSRF protections, integration guidance',
    category: 'Documentation',
    url: '/documentation/security',
    keywords: 'security ssrf sandbox local file access whitelist attack surface network trust',
    icon: 'sec',
  },
  {
    id: 'doc-performance',
    title: 'Performance & Profiling',
    desc: 'Memory benchmarks, CPU profiling, allocation metrics, process startup speed',
    category: 'Documentation',
    url: '/documentation/performance',
    keywords: 'performance benchmarks memory allocations cpu throughput profiling latency',
    icon: 'perf',
  },
  {
    id: 'doc-getting-started',
    title: 'Getting Started',
    desc: 'Installation, build flags, first CLI and Go library document generation',
    category: 'Documentation',
    url: '/getting-started',
    keywords: 'getting started installation quickstart build go install tutorial first pdf',
    icon: 'start',
  },
  {
    id: 'doc-about',
    title: 'About gowkhtmltopdf',
    desc: 'Project philosophy, MIT license, credits, and upstream wkhtmltopdf comparison',
    category: 'Documentation',
    url: '/about',
    keywords: 'about philosophy background author license credits history wkhtmltopdf',
    icon: 'about',
  },
  {
    id: 'doc-overview',
    title: 'Overview',
    desc: 'Landing page, print-ready document pipeline, key metrics and highlights',
    category: 'Documentation',
    url: '/',
    keywords: 'overview home landing intro hero features start',
    icon: 'home',
  },
]

const FLAG_ITEMS = [
  {
    id: 'flag-page-size',
    title: '--page-size <size>',
    desc: 'Set paper dimensions: A4, Letter, A3, Legal, Tabloid (default: A4)',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--page-size',
    keywords: 'page size paper format dimensions a4 letter a3 legal tabloid -s',
  },
  {
    id: 'flag-orientation',
    title: '--orientation <Portrait|Landscape>',
    desc: 'Set page orientation (default: Portrait)',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--orientation',
    keywords: 'orientation portrait landscape rotate direction -O',
  },
  {
    id: 'flag-margin-top',
    title: '--margin-top <unit>',
    desc: 'Set top page margin (e.g. 10mm, 0.5in, 20px) (default: 10mm)',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--margin-top',
    keywords: 'margin top spacing -T mm in px dimension',
  },
  {
    id: 'flag-margin-bottom',
    title: '--margin-bottom <unit>',
    desc: 'Set bottom page margin (e.g. 10mm, 0.5in, 20px) (default: 10mm)',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--margin-bottom',
    keywords: 'margin bottom spacing -B mm in px dimension',
  },
  {
    id: 'flag-margin-left',
    title: '--margin-left <unit>',
    desc: 'Set left page margin (e.g. 10mm, 0.5in, 20px) (default: 10mm)',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--margin-left',
    keywords: 'margin left spacing -L mm in px dimension',
  },
  {
    id: 'flag-margin-right',
    title: '--margin-right <unit>',
    desc: 'Set right page margin (e.g. 10mm, 0.5in, 20px) (default: 10mm)',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--margin-right',
    keywords: 'margin right spacing -R mm in px dimension',
  },
  {
    id: 'flag-enable-local-file-access',
    title: '--enable-local-file-access',
    desc: 'Allow reading local filesystem assets like images, stylesheets, and fonts',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--enable-local-file-access',
    keywords: 'security local files images assets fs read disk filesystem access allow',
  },
  {
    id: 'flag-allow',
    title: '--allow <path>',
    desc: 'Whitelist a specific directory path for local resource loading',
    category: 'CLI Flags',
    url: '/documentation/security',
    flag: '--allow',
    keywords: 'security allow whitelist path directory resource sandboxing access',
  },
  {
    id: 'flag-grayscale',
    title: '--grayscale / -g',
    desc: 'Render output PDF or raster images in grayscale mode',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--grayscale',
    keywords: 'grayscale color black and white print mono monochrome -g',
  },
  {
    id: 'flag-zoom',
    title: '--zoom <factor>',
    desc: 'Scale HTML layout zoom factor (e.g. 1.0, 1.25, 0.8)',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--zoom',
    keywords: 'zoom scale magnification html view size magnify',
  },
  {
    id: 'flag-copies',
    title: '--copies <n>',
    desc: 'Specify number of printed copies in the output PDF (default: 1)',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--copies',
    keywords: 'copies print number -c count replicate',
  },
  {
    id: 'flag-title',
    title: '--title <text>',
    desc: 'Set the PDF document metadata title',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--title',
    keywords: 'title metadata pdf document name -t header',
  },
  {
    id: 'flag-header-left',
    title: '--header-left / --header-right <text>',
    desc: 'Set running text headers with replacement tokens ([page], [title], [date])',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--header-left',
    keywords: 'header running header token page doctitle title date time text',
  },
  {
    id: 'flag-footer-center',
    title: '--footer-center <text>',
    desc: 'Set running pagination footer (e.g. "[page] / [topage]")',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--footer-center',
    keywords: 'footer running footer pagination page topage center number',
  },
  {
    id: 'flag-toc-header-text',
    title: '--toc-header-text <text>',
    desc: 'Set title text for generated Table of Contents',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--toc-header-text',
    keywords: 'toc table of contents header outline text title index',
  },
  {
    id: 'flag-outline',
    title: '--outline / --outline-depth <n>',
    desc: 'Generate PDF document bookmarks / outline tree (default: depth 4)',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--outline',
    keywords: 'outline bookmarks pdf tree navigation table contents depth',
  },
  {
    id: 'flag-font-path',
    title: '--font-path <dir>',
    desc: 'Specify custom directory for TTF / OTF font discovery and loading',
    category: 'CLI Flags',
    url: '/documentation/fonts',
    flag: '--font-path',
    keywords: 'font path directory ttf otf load discovery truetype opentype',
  },
  {
    id: 'flag-quiet',
    title: '--quiet / -q',
    desc: 'Suppress converter progress logging and information messages',
    category: 'CLI Flags',
    url: '/documentation/cli',
    flag: '--quiet',
    keywords: 'quiet silence suppress logs verbose output -q silent',
  },
]

const DOSSIER_BENCH_ITEMS = [
  {
    id: 'bench-overview',
    title: 'Benchmark Comparison',
    desc: 'Process speed and RSS memory comparison against wkhtmltopdf 0.12.6.1',
    category: 'Benchmarks & Dossier',
    url: '/benchmarks',
    keywords: 'benchmarks performance wkhtmltopdf comparison speed memory rss speedup latency',
    icon: 'bench',
  },
  {
    id: 'dossier-all',
    title: 'Issue Dossier (1,329 upstream issues)',
    desc: 'Comprehensive mapping of upstream wkhtmltopdf issues to status and coverage',
    category: 'Benchmarks & Dossier',
    url: '/dossier',
    keywords: 'dossier issues coverage 1329 github wkhtmltopdf bugs fixes status tracker',
    icon: 'dossier',
  },
  {
    id: 'dossier-css',
    title: 'Dossier: CSS & Layout Issues',
    desc: 'Flexbox, table pagination, margins, floats, and container queries coverage',
    category: 'Benchmarks & Dossier',
    url: '/dossier?cat=CSS%2Flayout',
    keywords: 'dossier css layout flexbox tables margins page breaks floats queries',
    icon: 'dossier',
  },
  {
    id: 'dossier-fonts',
    title: 'Dossier: Font & Text Issues',
    desc: 'OpenType, UTF-8, CJK fonts, font fallback, and text decoration coverage',
    category: 'Benchmarks & Dossier',
    url: '/dossier?cat=Fonts%2Fencoding%2Ftext',
    keywords: 'dossier fonts text cjk utf-8 encoding fallback opentype unicode',
    icon: 'dossier',
  },
  {
    id: 'dossier-crash',
    title: 'Dossier: Crash & Memory Issues',
    desc: 'Segmentation faults, infinite loops, and memory leak prevention',
    category: 'Benchmarks & Dossier',
    url: '/dossier?cat=Crash%2Fhang%2Fmemory',
    keywords: 'dossier crash segfault memory leak hang infinite loop stability',
    icon: 'dossier',
  },
]

export default function CommandPalette() {
  const [isOpen, setIsOpen] = useState(false)
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [toastMessage, setToastMessage] = useState(null)
  const [theme, setTheme] = useTheme()
  const navigate = useNavigate()

  const inputRef = useRef(null)
  const listRef = useRef(null)
  const previousActiveElement = useRef(null)

  // Build Showcase items list from data
  const showcaseItems = useMemo(() => {
    const allShowcase = [...SHOWCASE, ...SHOWCASE_SPECIAL]
    return allShowcase.map((item) => ({
      id: `showcase-${item.name}`,
      title: item.title,
      desc: `${item.desc} (${item.pages ?? 1} ${(item.pages ?? 1) === 1 ? 'page' : 'pages'})`,
      category: 'Showcase Samples',
      subcategory: item.category || 'Showcase',
      url: `/showcase?cat=${encodeURIComponent(item.category || 'All')}`,
      keywords: `showcase sample ${item.name} ${item.file} ${item.category || ''} ${item.title} invoice report poster storybook`,
      icon: 'sample',
    }))
  }, [])

  // Quick actions
  const actionItems = useMemo(() => [
    {
      id: 'action-theme',
      title: `Toggle Theme (Currently ${theme === 'dark' ? 'Dark' : 'Light'})`,
      desc: `Switch color scheme to ${theme === 'dark' ? 'Light' : 'Dark'} mode`,
      category: 'Quick Actions',
      keywords: 'theme dark light mode appearance color scheme switch toggle style',
      icon: 'theme',
      action: () => {
        const nextTheme = theme === 'dark' ? 'light' : 'dark'
        setTheme(nextTheme)
        showToast(`Theme switched to ${nextTheme} mode`)
      },
    },
    {
      id: 'action-github',
      title: 'View on GitHub',
      desc: 'Open chinmay-sawant/gowkhtmltopdf repository on GitHub',
      category: 'Quick Actions',
      keywords: 'github source code repo git repository upstream open web',
      icon: 'github',
      action: () => {
        window.open('https://github.com/chinmay-sawant/gowkhtmltopdf', '_blank', 'noopener,noreferrer')
      },
    },
    {
      id: 'action-copy-install',
      title: 'Copy Go Install Command',
      desc: 'go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.2',
      category: 'Quick Actions',
      keywords: 'install go binary cli copy command download build latest v0.2.2',
      icon: 'copy',
      action: () => {
        navigator.clipboard.writeText('go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.2')
        showToast('Install command copied to clipboard!')
      },
    },
  ], [theme, setTheme])

  // Combine all items into a master search index
  const masterIndex = useMemo(() => {
    return [
      ...DOC_ITEMS,
      ...FLAG_ITEMS,
      ...showcaseItems,
      ...DOSSIER_BENCH_ITEMS,
      ...actionItems,
    ]
  }, [showcaseItems, actionItems])

  const showToast = (msg) => {
    setToastMessage(msg)
    setTimeout(() => {
      setToastMessage(null)
    }, 2500)
  }

  // Open / Close listeners (Cmd+K / Ctrl+K and custom event)
  useEffect(() => {
    function handleKeyDown(e) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setIsOpen((prev) => !prev)
      }
    }

    function handleOpenEvent() {
      setIsOpen(true)
    }

    window.addEventListener('keydown', handleKeyDown)
    window.addEventListener('open-command-palette', handleOpenEvent)

    return () => {
      window.removeEventListener('keydown', handleKeyDown)
      window.removeEventListener('open-command-palette', handleOpenEvent)
    }
  }, [])

  // Focus management & body scroll lock
  useEffect(() => {
    if (isOpen) {
      previousActiveElement.current = document.activeElement
      document.body.style.overflow = 'hidden'
      setQuery('')
      setSelectedIndex(0)
      requestAnimationFrame(() => {
        inputRef.current?.focus()
      })
    } else {
      document.body.style.overflow = ''
      if (previousActiveElement.current && typeof previousActiveElement.current.focus === 'function') {
        previousActiveElement.current.focus()
      }
    }
  }, [isOpen])

  // Filtered and ranked results
  const filteredResults = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) {
      // Default curated suggestions when search query is empty
      return [
        ...actionItems,
        ...DOC_ITEMS.slice(0, 5),
        ...FLAG_ITEMS.slice(0, 4),
        ...DOSSIER_BENCH_ITEMS.slice(0, 2),
        ...showcaseItems.slice(0, 3),
      ]
    }

    const terms = q.split(/\s+/).filter(Boolean)

    const scored = masterIndex.map((item) => {
      let score = 0
      const titleLower = item.title.toLowerCase()
      const descLower = item.desc.toLowerCase()
      const flagLower = (item.flag || '').toLowerCase()
      const categoryLower = item.category.toLowerCase()
      const subcatLower = (item.subcategory || '').toLowerCase()
      const kwLower = (item.keywords || '').toLowerCase()

      // Exact matches get massive boost
      if (titleLower === q || flagLower === q) score += 100
      if (titleLower.startsWith(q) || flagLower.startsWith(q)) score += 50
      if (flagLower.includes(q)) score += 35

      for (const term of terms) {
        if (titleLower.includes(term)) score += 20
        if (flagLower.includes(term)) score += 20
        if (subcatLower.includes(term)) score += 10
        if (categoryLower.includes(term)) score += 8
        if (descLower.includes(term)) score += 6
        if (kwLower.includes(term)) score += 4
      }

      return { item, score }
    })

    return scored
      .filter((entry) => entry.score > 0)
      .sort((a, b) => b.score - a.score)
      .map((entry) => entry.item)
      .slice(0, 30) // cap results for performance and clean UI
  }, [query, masterIndex, actionItems, showcaseItems])

  // Reset selected index when filtered results change
  useEffect(() => {
    setSelectedIndex(0)
  }, [query])

  // Execute selected item
  const handleSelect = (item) => {
    if (!item) return
    if (item.action) {
      item.action()
    } else if (item.url) {
      navigate(item.url)
    }
    setIsOpen(false)
  }

  // Keyboard navigation inside the palette
  const handleInputKeyDown = (e) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      if (filteredResults.length === 0) return
      setSelectedIndex((prev) => {
        const next = (prev + 1) % filteredResults.length
        scrollToItem(next)
        return next
      })
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      if (filteredResults.length === 0) return
      setSelectedIndex((prev) => {
        const next = (prev - 1 + filteredResults.length) % filteredResults.length
        scrollToItem(next)
        return next
      })
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (filteredResults[selectedIndex]) {
        handleSelect(filteredResults[selectedIndex])
      }
    } else if (e.key === 'Escape') {
      e.preventDefault()
      setIsOpen(false)
    } else if (e.key === 'Tab') {
      e.preventDefault()
      // Keep focus on input for keyboard navigation
      inputRef.current?.focus()
    }
  }

  const scrollToItem = (index) => {
    const element = document.getElementById(`palette-item-${index}`)
    if (element && listRef.current) {
      element.scrollIntoView({ block: 'nearest' })
    }
  }

  if (!isOpen) return null

  const isMac = typeof navigator !== 'undefined' && /Mac|iPod|iPhone|iPad/.test(navigator.platform)

  return (
    <div
      className="palette-backdrop"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          setIsOpen(false)
        }
      }}
      role="presentation"
    >
      <div
        className="palette-dialog"
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
      >
        <div className="palette-header">
          <svg
            className="palette-search-icon"
            viewBox="0 0 20 20"
            width="18"
            height="18"
            aria-hidden="true"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
          >
            <circle cx="8.5" cy="8.5" r="5.5" />
            <path d="M13 13l4 4" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            className="palette-input"
            placeholder="Search documentation, CLI flags, samples, actions…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleInputKeyDown}
            aria-autocomplete="list"
            aria-controls="palette-list"
            aria-activedescendant={
              filteredResults[selectedIndex] ? `palette-item-${selectedIndex}` : undefined
            }
          />
          {query && (
            <button
              type="button"
              className="palette-clear-btn"
              onClick={() => {
                setQuery('')
                inputRef.current?.focus()
              }}
              aria-label="Clear search query"
            >
              ×
            </button>
          )}
          <button
            type="button"
            className="palette-close-btn"
            onClick={() => setIsOpen(false)}
            aria-label="Close command palette (Esc)"
          >
            <kbd>ESC</kbd>
          </button>
        </div>

        {toastMessage && (
          <div className="palette-toast" role="status" aria-live="polite">
            <span>✓</span> {toastMessage}
          </div>
        )}

        <div className="palette-body" id="palette-list" ref={listRef} role="listbox">
          {filteredResults.length === 0 ? (
            <div className="palette-empty">
              <p className="palette-empty-title">No matching results found for &ldquo;{query}&rdquo;</p>
              <p className="palette-empty-hint">
                Try searching for CLI flags (e.g. <code>--page-size</code>), documentation sections (<code>security</code>, <code>fonts</code>), or showcase samples (<code>invoice</code>, <code>storybook</code>).
              </p>
            </div>
          ) : (
            <div className="palette-items">
              {filteredResults.map((item, index) => {
                const isSelected = index === selectedIndex
                return (
                  <div
                    key={item.id}
                    id={`palette-item-${index}`}
                    role="option"
                    aria-selected={isSelected}
                    className={`palette-item ${isSelected ? 'palette-item-selected' : ''}`}
                    onClick={() => handleSelect(item)}
                    onMouseEnter={() => setSelectedIndex(index)}
                  >
                    <div className="palette-item-icon" aria-hidden="true">
                      {item.flag ? (
                        <span className="palette-flag-symbol">--</span>
                      ) : item.icon === 'cli' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <polyline points="4 17 10 11 4 5" />
                          <line x1="12" y1="19" x2="20" y2="19" />
                        </svg>
                      ) : item.icon === 'api' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <polyline points="16 18 22 12 16 6" />
                          <polyline points="8 6 2 12 8 18" />
                        </svg>
                      ) : item.icon === 'arch' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <rect x="3" y="3" width="7" height="7" rx="1" />
                          <rect x="14" y="3" width="7" height="7" rx="1" />
                          <rect x="14" y="14" width="7" height="7" rx="1" />
                          <rect x="3" y="14" width="7" height="7" rx="1" />
                        </svg>
                      ) : item.icon === 'compat' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
                          <polyline points="14 2 14 8 20 8" />
                        </svg>
                      ) : item.icon === 'font' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <polyline points="4 7 4 4 20 4 20 7" />
                          <line x1="9" y1="20" x2="15" y2="20" />
                          <line x1="12" y1="4" x2="12" y2="20" />
                        </svg>
                      ) : item.icon === 'sec' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
                        </svg>
                      ) : item.icon === 'perf' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <circle cx="12" cy="12" r="10" />
                          <polyline points="12 6 12 12 16 14" />
                        </svg>
                      ) : item.icon === 'start' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <polygon points="5 3 19 12 5 21 5 3" />
                        </svg>
                      ) : item.icon === 'about' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <circle cx="12" cy="12" r="10" />
                          <line x1="12" y1="16" x2="12" y2="12" />
                          <line x1="12" y1="8" x2="12.01" y2="8" />
                        </svg>
                      ) : item.icon === 'home' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M3 9l9-7 9 7v11a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
                          <polyline points="9 22 9 12 15 12 15 22" />
                        </svg>
                      ) : item.icon === 'bench' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <line x1="18" y1="20" x2="18" y2="10" />
                          <line x1="12" y1="20" x2="12" y2="4" />
                          <line x1="6" y1="20" x2="6" y2="14" />
                        </svg>
                      ) : item.icon === 'dossier' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M9 11l3 3L22 4" />
                          <path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11" />
                        </svg>
                      ) : item.icon === 'sample' ? (
                        item.subcategory === 'Invoices & receipts' ? (
                          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                            <path d="M14 2H6a2 2 0 00-2 2v16l4-2 4 2 4-2 4 2V8z" />
                            <line x1="9" y1="9" x2="15" y2="9" />
                            <line x1="9" y1="13" x2="13" y2="13" />
                          </svg>
                        ) : item.subcategory === 'Reports & tables' ? (
                          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                            <rect x="3" y="3" width="18" height="18" rx="2" />
                            <line x1="3" y1="9" x2="21" y2="9" />
                            <line x1="9" y1="21" x2="9" y2="9" />
                          </svg>
                        ) : item.subcategory === 'Storybooks & posters' ? (
                          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                            <rect x="3" y="3" width="18" height="18" rx="2" />
                            <circle cx="8.5" cy="8.5" r="1.5" />
                            <polyline points="21 15 16 10 5 21" />
                          </svg>
                        ) : item.subcategory === 'CSS & layout fixtures' ? (
                          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                            <polygon points="12 2 2 7 12 12 22 7 12 2" />
                            <polyline points="2 17 12 22 22 17" />
                            <polyline points="2 12 12 17 22 12" />
                          </svg>
                        ) : item.subcategory === 'Architecture & API' ? (
                          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                            <rect x="2" y="2" width="8" height="8" rx="1" />
                            <rect x="14" y="2" width="8" height="8" rx="1" />
                            <rect x="8" y="14" width="8" height="8" rx="1" />
                            <line x1="6" y1="10" x2="12" y2="14" />
                            <line x1="18" y1="10" x2="12" y2="14" />
                          </svg>
                        ) : (
                          <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                            <path d="M13 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V9z" />
                            <polyline points="13 2 13 9 20 9" />
                          </svg>
                        )
                      ) : item.icon === 'theme' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <circle cx="12" cy="12" r="5" />
                          <line x1="12" y1="1" x2="12" y2="3" />
                          <line x1="12" y1="21" x2="12" y2="23" />
                          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
                          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
                          <line x1="1" y1="12" x2="3" y2="12" />
                          <line x1="21" y1="12" x2="23" y2="12" />
                          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
                          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
                        </svg>
                      ) : item.icon === 'github' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
                          <path d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.87 8.17 6.84 9.5.5.08.66-.23.66-.5v-1.69c-2.77.6-3.36-1.34-3.36-1.34-.46-1.16-1.11-1.47-1.11-1.47-.91-.62.07-.6.07-.6 1 .07 1.53 1.03 1.53 1.03.87 1.52 2.34 1.07 2.91.83.1-.65.35-1.09.63-1.34-2.22-.25-4.55-1.11-4.55-4.92 0-1.11.38-2 1.03-2.71-.1-.25-.45-1.29.1-2.64 0 0 .84-.27 2.75 1.02.79-.22 1.65-.33 2.5-.33.85 0 1.71.11 2.5.33 1.91-1.29 2.75-1.02 2.75-1.02.55 1.35.2 2.39.1 2.64.65.71 1.03 1.6 1.03 2.71 0 3.82-2.34 4.66-4.57 4.91.36.31.69.92.69 1.85V21c0 .27.16.59.67.5C19.14 20.16 22 16.42 22 12A10 10 0 0012 2z" />
                        </svg>
                      ) : item.icon === 'copy' ? (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                          <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
                        </svg>
                      ) : (
                        <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" strokeWidth="2">
                          <path d="M9 18l6-6-6-6" />
                        </svg>
                      )}
                    </div>
                    <div className="palette-item-content">
                      <div className="palette-item-top">
                        <span className="palette-item-title">{item.title}</span>
                        {item.subcategory && (
                          <span className="palette-item-badge">{item.subcategory}</span>
                        )}
                      </div>
                      <span className="palette-item-desc">{item.desc}</span>
                    </div>
                    <div className="palette-item-action">
                      <span className="palette-item-cat">{item.category}</span>
                      <span className="palette-item-enter" aria-hidden="true">↵</span>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        <div className="palette-footer">
          <div className="palette-shortcuts">
            <span className="palette-shortcut">
              <kbd className="palette-kbd">↑</kbd> <kbd className="palette-kbd">↓</kbd> navigate
            </span>
            <span className="palette-shortcut">
              <kbd className="palette-kbd">↵</kbd> select
            </span>
            <span className="palette-shortcut">
              <kbd className="palette-kbd">esc</kbd> close
            </span>
          </div>
          <div className="palette-platform-hint">
            <kbd className="palette-kbd">{isMac ? '⌘K' : 'Ctrl+K'}</kbd>
          </div>
        </div>
      </div>
    </div>
  )
}
