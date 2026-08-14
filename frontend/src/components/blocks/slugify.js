/**
 * Converts a string heading (possibly containing markdown formatting) into a URL-friendly anchor id.
 *
 * @param {string} text
 * @returns {string}
 */
export function slugify(text) {
  if (!text || typeof text !== 'string') return ''
  return text
    .replace(/[`*_~[\]()]/g, '') // remove markdown punctuation
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '') // remove special characters
    .replace(/[\s_-]+/g, '-') // collapse whitespace and underscores to hyphens
    .replace(/^-+|-+$/g, '') // trim hyphens
}
