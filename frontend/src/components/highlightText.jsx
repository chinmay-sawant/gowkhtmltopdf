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
