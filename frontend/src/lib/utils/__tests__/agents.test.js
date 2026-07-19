import { describe, it, expect } from 'vitest'
import { shortAgentID, isAgentOnline, pivotParentMap } from '../agents.js'

describe('shortAgentID', () => {
  it('returns first segment of a UUID', () => {
    expect(shortAgentID('abc123-def456-ghi789')).toBe('abc123')
  })

  it('returns whole string if no dash', () => {
    expect(shortAgentID('abc123')).toBe('abc123')
  })

  it('handles empty string', () => {
    expect(shortAgentID('')).toBe('')
  })
})

describe('isAgentOnline', () => {
  it('returns false for dead agents', () => {
    expect(isAgentOnline({ IsDead: true })).toBe(false)
    expect(isAgentOnline({ isDead: true })).toBe(false)
  })

  it('returns true for sessions', () => {
    expect(isAgentOnline({ _kind: 'session' })).toBe(true)
  })

  it('returns true for beacons that have not missed checkin', () => {
    const now = 1000
    const beacon = {
      _kind: 'beacon',
      NextCheckin: 1010,
      Interval: 5000000000,
      Jitter: 1000000000,
    }
    expect(isAgentOnline(beacon, now)).toBe(true)
  })

  it('returns false for beacons that missed checkin', () => {
    const now = 2000
    const beacon = {
      _kind: 'beacon',
      NextCheckin: 1000,
      Interval: 5000000000,
      Jitter: 1000000000,
    }
    expect(isAgentOnline(beacon, now)).toBe(false)
  })
})

describe('pivotParentMap', () => {
  it('returns empty map for null graph', () => {
    const map = pivotParentMap(null)
    expect(map.size).toBe(0)
  })

  it('builds parent-child relationships', () => {
    const graph = {
      Children: [
        {
          Session: { ID: 'child1' },
          Children: [],
        },
      ],
    }
    const map = pivotParentMap(graph)
    expect(map.size).toBe(0) // no parent for root-level entries
  })
})
