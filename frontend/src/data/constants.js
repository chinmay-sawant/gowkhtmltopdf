export const STATUS_ORDER = ['implemented', 'partial', 'not-implemented']

export const STATUS_META = {
  implemented: { label: 'Implemented', color: 'var(--ok-ink)', accent: 'var(--ok-bg)', text: 'var(--ok-ink)', rawColor: '#9BBF88' },
  partial: { label: 'Partial', color: 'var(--warn-ink)', accent: 'var(--warn-bg)', text: 'var(--warn-ink)', rawColor: '#E7CD80' },
  'not-implemented': { label: 'Not implemented', color: 'var(--bad-ink)', accent: 'var(--bad-bg)', text: 'var(--bad-ink)', rawColor: '#D89A8B' },
}

export const STATUS_CARD = {
  implemented: { stripe: 'var(--ok-ink)', bg: 'var(--ok-bg)', title: 'var(--ok-ink)', label: 'var(--ok-ink)' },
  partial: { stripe: 'var(--warn-ink)', bg: 'var(--warn-bg)', title: 'var(--warn-ink)', label: 'var(--warn-ink)' },
  'not-implemented': { stripe: 'var(--bad-ink)', bg: 'var(--bad-bg)', title: 'var(--bad-ink)', label: 'var(--bad-ink)' },
  unknown: { stripe: 'var(--line)', bg: 'var(--surface)', title: 'var(--ink)', label: 'var(--muted)' },
}

export const SEVERITY_ORDER = ['High', 'Medium', 'Low']

export const SEVERITY_META = {
  High: { bg: 'var(--bad-bg)', text: 'var(--bad-ink)', color: 'var(--bad-ink)' },
  Medium: { bg: 'var(--warn-bg)', text: 'var(--warn-ink)', color: 'var(--warn-ink)' },
  Low: { bg: 'var(--ok-bg)', text: 'var(--ok-ink)', color: 'var(--ok-ink)' },
}

export const CATEGORY_ORDER = [
  'CSS/layout',
  'JavaScript/interactive',
  'Images',
  'Fonts/encoding/text',
  'Page size/margins/header-footer',
  'Crash/hang/memory',
  'CLI/args',
  'Feature request',
  'Docs',
  'Other',
]

export const CATEGORY_COLOR = {
  'CSS/layout': '#E7CD80',
  'JavaScript/interactive': '#7FA8C9',
  Images: '#9BBF88',
  'Fonts/encoding/text': '#C9A6B5',
  'Page size/margins/header-footer': '#8FB8A0',
  'Crash/hang/memory': '#D89A8B',
  'CLI/args': '#B0A8D9',
  'Feature request': '#C4B58C',
  Docs: '#A8A8A8',
  Other: '#D1C2B5',
}
