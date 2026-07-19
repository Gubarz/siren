import { describe, expect, it } from 'vitest'
import { shouldShowNotification } from '../notifications.js'

describe('shouldShowNotification', () => {
  it('allows enabled notification types by default', () => {
    expect(shouldShowNotification({ enabled: true }, 'session-opened')).toBe(true)
  })

  it('mutes notifications when globally disabled', () => {
    expect(shouldShowNotification({ enabled: false }, 'session-opened')).toBe(false)
    expect(shouldShowNotification(null, 'session-opened')).toBe(false)
  })

  it('mutes explicitly disabled event types', () => {
    const prefs = { enabled: true, types: { 'session-opened': false } }

    expect(shouldShowNotification(prefs, 'session-opened')).toBe(false)
    expect(shouldShowNotification(prefs, 'session-closed')).toBe(true)
  })

  it('respects same-day do-not-disturb windows', () => {
    const prefs = { enabled: true, dnd: { enabled: true, start: '09:00', end: '17:00' } }

    expect(shouldShowNotification(prefs, 'event', at('2026-07-19T10:00:00'))).toBe(false)
    expect(shouldShowNotification(prefs, 'event', at('2026-07-19T17:00:00'))).toBe(true)
  })

  it('respects overnight do-not-disturb windows', () => {
    const prefs = { enabled: true, dnd: { enabled: true, start: '22:00', end: '08:00' } }

    expect(shouldShowNotification(prefs, 'event', at('2026-07-19T23:30:00'))).toBe(false)
    expect(shouldShowNotification(prefs, 'event', at('2026-07-19T07:59:00'))).toBe(false)
    expect(shouldShowNotification(prefs, 'event', at('2026-07-19T08:00:00'))).toBe(true)
  })

  it('does not mute when DND start and end are equal', () => {
    const prefs = { enabled: true, dnd: { enabled: true, start: '08:00', end: '08:00' } }

    expect(shouldShowNotification(prefs, 'event', at('2026-07-19T08:00:00'))).toBe(true)
  })
})

function at(value) {
  return new Date(value)
}
