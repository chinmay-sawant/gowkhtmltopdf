import issues from './issues.json'

export function sortIssues() {
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
