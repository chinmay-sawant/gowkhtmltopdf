import { useEffect } from 'react'

export default function PageTitle({ title }) {
  useEffect(() => {
    const base = 'wkhtmltopdf / gowkhtmltopdf'
    document.title = title ? `${title} - ${base}` : base
  }, [title])
  return null
}
