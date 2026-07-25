import { describe, it, expect } from 'vitest'
import { parseExpiry, getExpiryStatus, EXPIRY_STYLES } from '../certificates.js'

describe('parseExpiry', () => {
  it('parses server time format correctly', () => {
    const result = parseExpiry('2025-06-15 12:00:00 UTC-0700')
    expect(result).toBeInstanceOf(Date)
    expect(result.getUTCFullYear()).toBe(2025)
    expect(result.getUTCMonth()).toBe(5)  // June (0-indexed)
    expect(result.getUTCDate()).toBe(15)
  })

  it('returns null for unparseable strings', () => {
    expect(parseExpiry('Unknown (could not parse certificate)')).toBeNull()
    expect(parseExpiry('')).toBeNull()
    expect(parseExpiry(null)).toBeNull()
  })

  it('parses with different timezone offsets', () => {
    const result = parseExpiry('2025-06-15 12:00:00 UTC+0200')
    expect(result).toBeInstanceOf(Date)
  })
})

describe('getExpiryStatus', () => {
  it('returns expired for past dates', () => {
    const past = new Date('2020-01-01')
    const status = getExpiryStatus(past)
    expect(status.label).toBe('Expired')
    expect(status.variant).toBe('danger')
    expect(status.style).toBe(EXPIRY_STYLES.danger)
  })

  it('returns expiring_soon for dates within 7 days', () => {
    const sixDaysFromNow = new Date(Date.now() + 6 * 24 * 60 * 60 * 1000)
    const status = getExpiryStatus(sixDaysFromNow)
    expect(status.label).toBe('Expiring Soon')
    expect(status.variant).toBe('danger')
  })

  it('returns unknown for null or unparseable dates', () => {
    const status = getExpiryStatus(null)
    expect(status.label).toBe('Unknown')
    expect(status.variant).toBe('default')
    expect(status.style).toBe('')
    expect(status.relative).toBe('\u2014')
  })

  it('returns expiring for exactly 7 days from now', () => {
    const exactly7Days = new Date(Date.now() + 7 * 24 * 60 * 60 * 1000)
    const status = getExpiryStatus(exactly7Days)
    expect(status.label).toBe('Expiring')
    expect(status.variant).toBe('warning')
  })

  it('returns valid for exactly 30 days from now', () => {
    const exactly30Days = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000)
    const status = getExpiryStatus(exactly30Days)
    expect(status.label).toBe('Valid')
    expect(status.variant).toBe('success')
  })

  it('returns expiring for dates within 30 days but beyond 7 days', () => {
    const twentyDaysFromNow = new Date(Date.now() + 20 * 24 * 60 * 60 * 1000)
    const status = getExpiryStatus(twentyDaysFromNow)
    expect(status.label).toBe('Expiring')
    expect(status.variant).toBe('warning')
  })

  it('returns valid for dates beyond 30 days', () => {
    const ninetyDaysFromNow = new Date(Date.now() + 90 * 24 * 60 * 60 * 1000)
    const status = getExpiryStatus(ninetyDaysFromNow)
    expect(status.label).toBe('Valid')
    expect(status.variant).toBe('success')
  })

  it('returns relative time string for valid dates', () => {
    const ninetyDaysFromNow = new Date(Date.now() + 90 * 24 * 60 * 60 * 1000)
    const status = getExpiryStatus(ninetyDaysFromNow)
    expect(status.relative).toMatch(/expires in \d+ days?/)
  })

  it('returns relative time string for expired dates', () => {
    const past = new Date('2020-01-01')
    const status = getExpiryStatus(past)
    expect(status.relative).toMatch(/expired \d+ days? ago|expired today/)
  })
})

describe('EXPIRY_STYLES', () => {
  it('has entries for all status variants', () => {
    expect(EXPIRY_STYLES).toHaveProperty('danger')
    expect(EXPIRY_STYLES).toHaveProperty('warning')
    expect(EXPIRY_STYLES).toHaveProperty('success')
  })

  it('returns CSS custom property strings for non-success variants', () => {
    expect(EXPIRY_STYLES.danger).toContain('--')
    expect(EXPIRY_STYLES.warning).toContain('--')
    expect(EXPIRY_STYLES.success).toBe('')
  })
})
