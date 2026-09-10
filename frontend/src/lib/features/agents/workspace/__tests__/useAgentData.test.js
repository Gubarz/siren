import { describe, expect, it } from 'vitest'
import { createAgentDataModel } from '../useAgentData.svelte.js'

const liveSession = {
  ID: 'live-1',
  Name: 'live-agent',
  Hostname: 'LIVE-PC',
  Username: 'CORP\\live',
  OS: 'linux',
  RemoteAddress: '10.0.0.1:4444',
  Transport: 'mtls',
}

const liveBeacon = {
  ID: 'beacon-1',
  Name: 'beacon-agent',
  Hostname: 'BEACON-PC',
  Username: 'CORP\\beacon',
  OS: 'windows',
  RemoteAddress: '10.0.0.2:4444',
  Transport: 'http',
}

// Bindings deliver known agents in camelCase, matching KnownAgentView.
const lostRecord = {
  id: 'lost-1',
  kind: 'session',
  name: 'ghost',
  hostname: 'GONE-PC',
  username: 'CORP\\ghost',
  os: 'windows',
  arch: 'amd64',
  remoteAddress: '10.0.0.9:5555',
  transport: 'mtls',
  status: 'lost',
  firstSeen: 1000,
  lastSeen: 2000,
  lostAt: 3000,
}

function run({ sessions = [], beacons = [], known = [], filter = '', tags = {} } = {}) {
  const model = createAgentDataModel()
  return model.process(filter, sessions, beacons, [], null, tags, known)
}

describe('useAgentData lost-agent merge', () => {
  it('appends a lost record when no live agent shares its id', () => {
    const { combinedData, filteredData } = run({ sessions: [liveSession], known: [lostRecord] })

    expect(combinedData).toHaveLength(2)
    expect(combinedData[1]).toMatchObject({
      ID: 'lost-1',
      Name: 'ghost',
      Hostname: 'GONE-PC',
      Username: 'CORP\\ghost',
      OS: 'windows',
      RemoteAddress: '10.0.0.9:5555',
      Transport: 'mtls',
      _kind: 'session',
      _lost: true,
    })
    expect(filteredData).toHaveLength(2)
  })

  it('hides a lost record whose id matches a live session', () => {
    const lostLive = { ...lostRecord, id: liveSession.ID }
    const { combinedData } = run({ sessions: [liveSession], known: [lostLive] })

    expect(combinedData).toHaveLength(1)
    expect(combinedData[0]._lost).toBeUndefined()
  })

  it('hides a lost record whose id matches a live beacon', () => {
    const lostLive = { ...lostRecord, id: liveBeacon.ID }
    const { combinedData } = run({ beacons: [liveBeacon], known: [lostLive] })

    expect(combinedData).toHaveLength(1)
    expect(combinedData[0]._kind).toBe('beacon')
  })

  it('ignores known records that are not lost', () => {
    const { combinedData } = run({ known: [{ ...lostRecord, status: 'active' }] })
    expect(combinedData).toHaveLength(0)
  })

  it('keeps lost records out of the graph arrays', () => {
    const { graphSessions, graphBeacons } = run({
      sessions: [liveSession],
      beacons: [liveBeacon],
      known: [lostRecord],
    })

    expect(graphSessions.map((s) => s.ID)).toEqual([liveSession.ID])
    expect(graphBeacons.map((b) => b.ID)).toEqual([liveBeacon.ID])
  })

  it('returns no graph entries when only lost records exist', () => {
    const { graphSessions, graphBeacons } = run({ known: [lostRecord] })
    expect(graphSessions).toEqual([])
    expect(graphBeacons).toEqual([])
  })

  it('sorts lost records newest last-seen first after live entries', () => {
    const older = { ...lostRecord, id: 'lost-old', lastSeen: 1000 }
    const newer = { ...lostRecord, id: 'lost-new', lastSeen: 9000 }
    const { combinedData } = run({ sessions: [liveSession], known: [older, newer] })

    expect(combinedData.map((a) => a.ID)).toEqual([liveSession.ID, 'lost-new', 'lost-old'])
  })

  it('matches lost records with the same filter helper as live agents', () => {
    const { filteredData } = run({
      sessions: [liveSession],
      known: [lostRecord],
      filter: 'gone-pc',
    })

    expect(filteredData).toHaveLength(1)
    expect(filteredData[0].ID).toBe('lost-1')
  })

  it('matches lost records by tag', () => {
    const { filteredData } = run({
      known: [lostRecord],
      filter: 'archived',
      tags: { 'lost-1': ['archived'] },
    })

    expect(filteredData).toHaveLength(1)
    expect(filteredData[0]._lost).toBe(true)
  })
})
