import { useState, useEffect, useCallback } from 'react'
import { fetchIssues, sortIssues } from '../data/issues'

export function useIssues() {
  const [issues, setIssues] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const reload = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchIssues()
      setIssues(sortIssues(data))
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    reload()
  }, [reload])

  return { issues, loading, error, reload }
}

export default useIssues
