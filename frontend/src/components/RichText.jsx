import { highlightText } from './highlightText'

const CODE = /`([^`]+)`/g

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
