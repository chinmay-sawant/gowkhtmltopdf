let cachedIssues = null

export async function fetchIssues() {
  if (cachedIssues) return cachedIssues
  const basePath = import.meta.env.BASE_URL || '/'
  const url = `${basePath.endsWith('/') ? basePath : basePath + '/'}data/issues.json`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`Failed to load issue data (${res.status})`)
  const data = await res.json()
  cachedIssues = data
  return data
}

export function sortIssues(issues) {
  return [...issues].sort((a, b) => b.number - a.number)
}

export function countBy(issues, key) {
  const counts = {}
  for (const it of issues) {
    const k = it[key] ?? 'unknown'
    counts[k] = (counts[k] ?? 0) + 1
  }
  return counts
}
