const CODE = /`([^`]+)`/g

export default function RichText({ children }) {
  if (children == null || children === '') return null
  if (typeof children !== 'string') return children

  const parts = []
  let last = 0
  let key = 0
  let match
  const pattern = new RegExp(CODE.source, 'g')

  while ((match = pattern.exec(children)) !== null) {
    if (match.index > last) parts.push(children.slice(last, match.index))
    parts.push(<code key={key++}>{match[1]}</code>)
    last = pattern.lastIndex
  }

  if (parts.length === 0) return children
  if (last < children.length) parts.push(children.slice(last))
  return parts
}
