/**
 * Date and time formatting.
 *
 * Everything renders in the viewer's own locale and timezone. The backend
 * sends RFC3339 timestamps and YYYY-MM-DD dates, so the browser can do the
 * conversion — which matters because the API deliberately does not assume a
 * timezone for display.
 */

const DATE_ONLY = /^\d{4}-\d{2}-\d{2}$/

/** Parse an API date or timestamp into a Date, or null if unusable. */
export function parseApiDate(value) {
  if (!value) return null

  // A bare YYYY-MM-DD is parsed as UTC midnight by the Date constructor, which
  // renders as the previous day for anyone west of Greenwich. Constructing it
  // from parts keeps it on the intended calendar day everywhere.
  if (DATE_ONLY.test(value)) {
    const [year, month, day] = value.split('-').map(Number)
    return new Date(year, month - 1, day)
  }

  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

export function formatDate(value) {
  const date = parseApiDate(value)
  if (!date) return '—'
  return date.toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' })
}

export function formatTime(value) {
  const date = parseApiDate(value)
  if (!date) return '—'
  return date.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })
}

export function formatDateTime(value) {
  const date = parseApiDate(value)
  if (!date) return '—'
  return date.toLocaleString(undefined, {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** True when the timestamp falls on the viewer's current local day. */
export function isToday(value) {
  const date = parseApiDate(value)
  if (!date) return false

  const now = new Date()
  return (
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate()
  )
}

/** Whole days elapsed since a date, floored at zero. */
export function daysSince(value) {
  const date = parseApiDate(value)
  if (!date) return null

  const millisPerDay = 24 * 60 * 60 * 1000
  const startOfThen = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const now = new Date()
  const startOfNow = new Date(now.getFullYear(), now.getMonth(), now.getDate())

  return Math.max(0, Math.round((startOfNow - startOfThen) / millisPerDay))
}

/** Initials for an avatar, e.g. "Jane Otieno" -> "JO". */
export function initials(fullName) {
  if (!fullName) return '?'
  return fullName
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join('')
}
