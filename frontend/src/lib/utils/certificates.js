export function parseExpiry(raw) {
  if (!raw || typeof raw !== 'string') return null
  const str = raw.trim()
  if (!str || str.startsWith('Unknown')) return null
  const cleaned = str.replace(' UTC', '').replace(' ', 'T').replace(/([+-]\d{2})(\d{2})$/, '$1:$2')
  const date = new Date(cleaned)
  if (isNaN(date.getTime())) return null
  return date
}

const ONE_WEEK = 7 * 24 * 60 * 60 * 1000
const ONE_MONTH = 30 * 24 * 60 * 60 * 1000

export function getExpiryStatus(expiryDate) {
  if (!expiryDate || isNaN(expiryDate.getTime())) {
    return {
      label: 'Unknown',
      variant: 'default',
      style: '',
      relative: '\u2014',
    }
  }

  const now = Date.now()
  const expiry = expiryDate.getTime()

  if (expiry <= now) {
    return {
      label: 'Expired',
      variant: 'danger',
      style: EXPIRY_STYLES.danger,
      relative: relativeTime(expiry, now, true),
    }
  }

  if (expiry < now + ONE_WEEK) {
    return {
      label: 'Expiring Soon',
      variant: 'danger',
      style: EXPIRY_STYLES.danger,
      relative: relativeTime(expiry, now, false),
    }
  }

  if (expiry < now + ONE_MONTH) {
    return {
      label: 'Expiring',
      variant: 'warning',
      style: EXPIRY_STYLES.warning,
      relative: relativeTime(expiry, now, false),
    }
  }

  return {
    label: 'Valid',
    variant: 'success',
    style: EXPIRY_STYLES.success,
    relative: relativeTime(expiry, now, false),
  }
}

function relativeTime(expiry, now, isExpired) {
  const diffMs = isExpired ? now - expiry : expiry - now
  const days = Math.floor(diffMs / (24 * 60 * 60 * 1000))
  if (isExpired) {
    if (days <= 0) return 'expired today'
    return `expired ${days} day${days !== 1 ? 's' : ''} ago`
  }
  if (days <= 0) return 'expires today'
  return `expires in ${days} day${days !== 1 ? 's' : ''}`
}

export const EXPIRY_STYLES = {
  danger: 'background-color: color-mix(in srgb, var(--color-danger-500) 12%, transparent); color: var(--color-danger-400);',
  warning: 'background-color: color-mix(in srgb, var(--color-warning-500) 12%, transparent); color: var(--color-warning-400);',
  success: '',  // no override — use default row style
}
