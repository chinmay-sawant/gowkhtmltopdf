const CODE = /`([^`]+)`/g

function escapeRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function highlightText(text, query, keyPrefix = 'hl') {
  if (!query || typeof text !== 'string' || !text) return text
  const rawQ = query.trim()
  if (!rawQ) return text

  const cleanQ = rawQ.startsWith('#') ? rawQ.slice(1).trim() : ''
  const terms = Array.from(new Set([rawQ, cleanQ].filter((t) => t.length > 0)))
  if (terms.length === 0) return text

  const regexPattern = terms.map(escapeRegex).join('|')
  if (!regexPattern) return text

  const regex = new RegExp(`(${regexPattern})`, 'gi')
  const parts = text.split(regex)
  if (parts.length <= 1) return text

  return parts.map((part, i) => {
    if (regex.test(part)) {
      regex.lastIndex = 0
      return (
        <mark key={`${keyPrefix}-${i}`} className="search-highlight">
          {part}
        </mark>
      )
    }
    return part
  })
}

export default function RichText({ children, highlight }) {
  if (children == null || children === '') return null
  if (typeof children !== 'string') return children

  const parts = []
  let last = 0
  let key = 0
  let match
  const pattern = new RegExp(CODE.source, 'g')

  while ((match = pattern.exec(children)) !== null) {
    if (match.index > last) {
      const textChunk = children.slice(last, match.index)
      parts.push(highlight ? highlightText(textChunk, highlight, `txt-${key++}`) : textChunk)
    }
    const codeContent = match[1]
    parts.push(
      <code key={`code-${key++}`}>
        {highlight ? highlightText(codeContent, highlight, `cd-${key}`) : codeContent}
      </code>
    )
    last = pattern.lastIndex
  }

  if (parts.length === 0) {
    return highlight ? highlightText(children, highlight, 'root') : children
  }
  if (last < children.length) {
    const remChunk = children.slice(last)
    parts.push(highlight ? highlightText(remChunk, highlight, `txt-${key++}`) : remChunk)
  }
  return parts
}
