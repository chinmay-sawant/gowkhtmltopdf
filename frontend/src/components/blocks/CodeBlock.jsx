import { useState, useRef, useEffect } from 'react'
import { slugify } from './slugify'

export default function CodeBlock({ block }) {
  const [copied, setCopied] = useState(false)
  const timerRef = useRef(null)

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current)
      }
    }
  }, [])

  const handleCopy = async () => {
    try {
      if (navigator?.clipboard?.writeText) {
        await navigator.clipboard.writeText(block.code)
      } else {
        const textarea = document.createElement('textarea')
        textarea.value = block.code
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        document.execCommand('copy')
        document.body.removeChild(textarea)
      }
      setCopied(true)
      if (timerRef.current) clearTimeout(timerRef.current)
      timerRef.current = setTimeout(() => {
        setCopied(false)
      }, 2000)
    } catch (err) {
      console.error('Failed to copy text:', err)
    }
  }

  return (
    <section className="code-block">
      {block.heading && (
        <h3 className="code-block-heading" id={slugify(block.heading)}>
          {block.heading}
        </h3>
      )}
      <div className="code-block-wrapper">
        <button
          type="button"
          className={`code-copy-btn ${copied ? 'copied' : ''}`}
          onClick={handleCopy}
          aria-label={copied ? 'Copied code to clipboard' : 'Copy code to clipboard'}
          title={copied ? 'Copied to clipboard' : 'Copy code'}
        >
          {copied ? (
            <>
              <svg
                className="code-copy-icon"
                width="13"
                height="13"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <polyline points="20 6 9 17 4 12" />
              </svg>
              <span>Copied!</span>
            </>
          ) : (
            <>
              <svg
                className="code-copy-icon"
                width="13"
                height="13"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
                aria-hidden="true"
              >
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
              </svg>
              <span>Copy</span>
            </>
          )}
        </button>
        <pre className={`lang-${block.language}`}>
          <code>{block.code}</code>
        </pre>
      </div>
    </section>
  )
}
