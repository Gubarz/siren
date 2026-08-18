import { describe, expect, it } from 'vitest'
import { formatBytes, formatDateTime, formatRelativeTime, implantFormat } from '../formats.js'

describe('implantFormat', () => {
  it('maps numeric formats to labels', () => {
    expect(implantFormat(0)).toBe('shared lib')
    expect(implantFormat(1)).toBe('shellcode')
    expect(implantFormat(2)).toBe('executable')
    expect(implantFormat(3)).toBe('service')
    expect(implantFormat(4)).toBe('third-party')
  })

  it('passes through unknown values', () => {
    expect(implantFormat(9)).toBe(9)
    expect(implantFormat(undefined)).toBe(undefined)
  })

  it('passes through string values untouched', () => {
    expect(implantFormat('executable')).toBe('executable')
    expect(implantFormat('')).toBe('')
  })
})

describe('formatBytes', () => {
  it('formats zero and negative values with default and custom zeroText', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(null)).toBe('0 B')
    expect(formatBytes(undefined, { zeroText: '-' })).toBe('-')
    expect(formatBytes(-100, { zeroText: '-' })).toBe('-')
  })

  it('formats bytes, kilobytes, and megabytes', () => {
    expect(formatBytes(500)).toBe('500 B')
    expect(formatBytes(1024)).toBe('1 KB')
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatBytes(1048576)).toBe('1 MB')
    expect(formatBytes(1073741824)).toBe('1 GB')
  })

  it('supports binary IEC units (KiB, MiB)', () => {
    expect(formatBytes(1024, { binaryUnits: true })).toBe('1 KiB')
    expect(formatBytes(2097152, { binaryUnits: true })).toBe('2 MiB')
  })

  it('supports custom decimal precision', () => {
    expect(formatBytes(1234567, { decimals: 2 })).toBe('1.18 MB')
  })
})

describe('formatRelativeTime', () => {
  const now = 1700000000

  it('returns - for empty or invalid values', () => {
    expect(formatRelativeTime(null)).toBe('-')
    expect(formatRelativeTime(0)).toBe('-')
    expect(formatRelativeTime('invalid')).toBe('-')
  })

  it('formats seconds, minutes, hours, days', () => {
    expect(formatRelativeTime(now - 1, now)).toBe('just now')
    expect(formatRelativeTime(now - 30, now)).toBe('30s ago')
    expect(formatRelativeTime(now - 300, now)).toBe('5m ago')
    expect(formatRelativeTime(now - 7200, now)).toBe('2h ago')
    expect(formatRelativeTime(now - 172800, now)).toBe('2d ago')
  })

  it('handles millisecond timestamps and ISO strings', () => {
    expect(formatRelativeTime((now - 45) * 1000, now)).toBe('45s ago')
    const iso = new Date((now - 120) * 1000).toISOString()
    expect(formatRelativeTime(iso, now)).toBe('2m ago')
  })
})

describe('formatDateTime', () => {
  it('returns - for empty values', () => {
    expect(formatDateTime(null)).toBe('-')
    expect(formatDateTime('')).toBe('-')
  })

  it('formats valid dates from seconds, ms, or ISO strings', () => {
    expect(formatDateTime(1700000000)).not.toBe('-')
    expect(formatDateTime('2026-01-01T12:00:00Z')).not.toBe('-')
  })
})
