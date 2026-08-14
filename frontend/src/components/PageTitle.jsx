import { useEffect } from 'react'

export default function PageTitle({ title }) {
  useEffect(() => {
    document.title = title
      ? `${title} — gowkhtmltopdf`
      : 'gowkhtmltopdf: print-oriented HTML rendering in Go'
  }, [title])
  return null
}

