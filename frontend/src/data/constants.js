export const STATUS_ORDER = ['implemented', 'partial', 'not-implemented', 'unassessed']

export const STATUS_META = {
  implemented: { label: 'Implemented', color: '#9BBF88', accent: '#EDF3EC', text: '#346538' },
  partial: { label: 'Partial', color: '#E7CD80', accent: '#FBF3DB', text: '#956400' },
  'not-implemented': { label: 'Not implemented', color: '#D89A8B', accent: '#FDEBEC', text: '#9F2F2D' },
  unassessed: { label: 'Unassessed', color: '#A8A8A8', accent: '#F0EFEA', text: '#5F5F5C' },
}

export const STATUS_CARD = {
  implemented: { stripe: '#9BBF88', bg: '#F6FAF5', title: '#346538', label: '#346538' },
  partial: { stripe: '#E7CD80', bg: '#FCF9F1', title: '#7A5A05', label: '#956400' },
  'not-implemented': { stripe: '#D89A8B', bg: '#FBF4F2', title: '#8A3A2E', label: '#9F2F2D' },
  unassessed: { stripe: '#C4C4C0', bg: '#FAFAF8', title: '#4A4A47', label: '#5F5F5C' },
  unknown: { stripe: '#EAEAEA', bg: '#FFFFFF', title: '#2F3437', label: '#787774' },
}

export const SEVERITY_ORDER = ['High', 'Medium', 'Low']

export const SEVERITY_META = {
  High: { bg: '#FDEBEC', text: '#9F2F2D' },
  Medium: { bg: '#FBF3DB', text: '#956400' },
  Low: { bg: '#EDF3EC', text: '#346538' },
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
