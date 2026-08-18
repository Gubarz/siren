import { describe, it, expect } from 'vitest'
import {
  shortAgentID,
  isAgentOnline,
  pivotParentMap,
  buildAgentMap,
  agentKind,
  osIcon,
  collectAgents,
} from '../agents.js'

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

describe('buildAgentMap', () => {
  it('indexes sessions and beacons by ID, annotated with _kind', () => {
    const map = buildAgentMap([{ ID: 's-1', Hostname: 'ws1' }], [{ ID: 'b-1', Hostname: 'ws2' }])
    expect(map.get('s-1')).toMatchObject({ ID: 's-1', Hostname: 'ws1', _kind: 'session' })
    expect(map.get('b-1')).toMatchObject({ ID: 'b-1', Hostname: 'ws2', _kind: 'beacon' })
  })

  it('does not mutate the source objects', () => {
    const session = { ID: 's-1' }
    buildAgentMap([session], [])
    expect(session._kind).toBeUndefined()
  })

  it('lets beacons win on ID collision (matches dropdown lookup order)', () => {
    const map = buildAgentMap([{ ID: 'x', Hostname: 'as-session' }], [{ ID: 'x', Hostname: 'as-beacon' }])
    expect(map.get('x')).toMatchObject({ Hostname: 'as-beacon', _kind: 'beacon' })
  })

  it('tolerates null/undefined lists', () => {
    expect(buildAgentMap(null, undefined).size).toBe(0)
  })
})

describe('agentKind', () => {
  it('identifies session and beacon objects correctly', () => {
    expect(agentKind({ _kind: 'session' })).toBe('session')
    expect(agentKind({ _kind: 'beacon' })).toBe('beacon')
    expect(agentKind({ NextCheckin: 12345 })).toBe('beacon')
    expect(agentKind({ ID: 's-1' })).toBe('session')
    expect(agentKind(null)).toBe('session')
  })
})

describe('osIcon', () => {
  it('maps common operating system strings to icons', () => {
    expect(osIcon('windows')).toBe('monitor')
    expect(osIcon('win10')).toBe('monitor')
    expect(osIcon('linux')).toBe('terminal')
    expect(osIcon('darwin')).toBe('apple')
    expect(osIcon('macOS')).toBe('apple')
    expect(osIcon('android')).toBe('smartphone')
    expect(osIcon('ios')).toBe('smartphone')
    expect(osIcon('unknown')).toBe('cpu')
    expect(osIcon('')).toBe('cpu')
  })
})

describe('collectAgents', () => {
  it('merges sessions and beacons into a unified array tagged with _kind', () => {
    const list = collectAgents({
      sessions: [{ ID: 's-1' }],
      beacons: [{ ID: 'b-1' }],
    })
    expect(list).toEqual([
      { ID: 's-1', _kind: 'session' },
      { ID: 'b-1', _kind: 'beacon' },
    ])
  })

  it('handles empty input gracefully', () => {
    expect(collectAgents()).toEqual([])
  })
})
