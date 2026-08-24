import { describe, expect, it, vi, beforeAll, beforeEach, afterEach } from 'vitest'

const runtime = vi.hoisted(() => ({
  onBloodhoundEvent: vi.fn(),
}))
vi.mock('$api/runtime.js', () => runtime)

const api = vi.hoisted(() => ({
  correlateAgents: vi.fn(),
  getBloodHoundStatus: vi.fn(() => Promise.resolve(null)),
  getBloodHoundIngestJobs: vi.fn(() => Promise.resolve([])),
  getBloodHoundCollections: vi.fn(() => Promise.resolve([])),
}))
vi.mock('$api/bloodhound.js', () => api)

import { bloodhoundStore, subscribeBloodhound, requestCorrelation, pullBloodhoundState } from '../bloodhound.svelte.js'
import { enrichmentChip, pathTitle } from '../enrichment.js'

let eventHandler = null

describe('bloodhoundStore', () => {
  beforeAll(() => {
    runtime.onBloodhoundEvent.mockImplementation((cb) => { eventHandler = cb })
    subscribeBloodhound()
  })

  beforeEach(() => {
    vi.clearAllMocks()
    bloodhoundStore.status = null
    bloodhoundStore.connected = false
    bloodhoundStore.domains = []
    bloodhoundStore.enrichment = {}
    bloodhoundStore.ingestJobs = []
    bloodhoundStore.collections = []
    bloodhoundStore.collectionRequest = null
  })

  afterEach(() => vi.useRealTimers())

  it('routes status events', () => {
    subscribeBloodhound()
    eventHandler({ type: 'bloodhound.status', payload: { connected: true, serverUrl: 'https://bh' } })
    expect(bloodhoundStore.connected).toBe(true)
    expect(bloodhoundStore.status).toMatchObject({ connected: true })
  })

  it('pulls current status, jobs, and collections', async () => {
    api.getBloodHoundStatus.mockResolvedValue({ connected: true, serverUrl: 'https://bh' })
    api.getBloodHoundIngestJobs.mockResolvedValue([{ id: 1 }])
    api.getBloodHoundCollections.mockResolvedValue([{ id: 'run1' }])

    await pullBloodhoundState()

    expect(bloodhoundStore.connected).toBe(true)
    expect(bloodhoundStore.status).toMatchObject({ connected: true, serverUrl: 'https://bh' })
    expect(bloodhoundStore.ingestJobs).toEqual([{ id: 1 }])
    expect(bloodhoundStore.collections).toEqual([{ id: 'run1' }])
  })

  it('routes synced events', () => {
    subscribeBloodhound()
    eventHandler({
      type: 'bloodhound.synced',
      payload: {
        domains: [{ objectId: 'D1', name: 'corp.local' }],
        enrichments: { a1: { entity: { objectId: 'S-1' }, distanceToTierZero: 1 } },
      },
    })
    expect(bloodhoundStore.domains).toHaveLength(1)
    expect(bloodhoundStore.enrichment.a1).toMatchObject({ distanceToTierZero: 1 })
  })

  it('routes enrichment events', () => {
    subscribeBloodhound()
    eventHandler({ type: 'bloodhound.enrichment', payload: { a1: { owned: true } } })
    expect(bloodhoundStore.enrichment.a1).toMatchObject({ owned: true })
  })

  it('ignores unknown event types', () => {
    subscribeBloodhound()
    eventHandler({ type: 'sliver.something', payload: {} })
    expect(bloodhoundStore.connected).toBe(false)
    expect(bloodhoundStore.enrichment).toEqual({})
  })

  it('tracks ingest job lifecycle events', () => {
    subscribeBloodhound()
    eventHandler({ type: 'bloodhound.ingest.job.started', payload: { id: 7, status: 'ready' } })
    expect(bloodhoundStore.ingestJobs).toHaveLength(1)
    expect(bloodhoundStore.ingestJobs[0]).toMatchObject({ id: 7, status: 'ready' })

    eventHandler({ type: 'bloodhound.ingest.job.completed', payload: { id: 7, status: 'complete', totalFiles: 2 } })
    expect(bloodhoundStore.ingestJobs).toHaveLength(1)
    expect(bloodhoundStore.ingestJobs[0]).toMatchObject({ id: 7, status: 'complete', totalFiles: 2 })

    eventHandler({ type: 'bloodhound.ingest.job.started', payload: { id: 8, status: 'ready' } })
    expect(bloodhoundStore.ingestJobs).toHaveLength(2)
    expect(bloodhoundStore.ingestJobs[0].id).toBe(8)
  })

  it('tracks collection stage events by run id', () => {
    subscribeBloodhound()
    eventHandler({ type: 'bloodhound.collection.run1.staged', payload: { id: 'run1', agentId: 'a1', stage: 'staged' } })
    eventHandler({ type: 'bloodhound.collection.run1.collecting', payload: { id: 'run1', agentId: 'a1', stage: 'collecting' } })
    eventHandler({ type: 'bloodhound.collection.run2.staged', payload: { id: 'run2', agentId: 'a1', stage: 'staged' } })
    eventHandler({ type: 'bloodhound.collection.run1.failed', payload: { id: 'run1', agentId: 'a1', stage: 'failed', error: 'boom' } })

    expect(bloodhoundStore.collections).toHaveLength(2)
    const run1 = bloodhoundStore.collections.find((c) => c.id === 'run1')
    expect(run1).toMatchObject({ stage: 'failed', error: 'boom' })
    expect(bloodhoundStore.collections.find((c) => c.id === 'run2').stage).toBe('staged')
  })

  it('debounces correlation requests into one call', () => {
    vi.useFakeTimers()
    subscribeBloodhound()
    eventHandler({ type: 'bloodhound.status', payload: { connected: true } })
    api.correlateAgents.mockResolvedValue({ a1: { owned: true } })

    const agents = [{ ID: 'a1', Hostname: 'pc1', Username: 'CORP\\jane', RemoteAddress: '10.0.0.1' }]
    requestCorrelation(agents)
    requestCorrelation(agents)
    expect(api.correlateAgents).not.toHaveBeenCalled()

    vi.advanceTimersByTime(500)
    expect(api.correlateAgents).toHaveBeenCalledTimes(1)
    expect(api.correlateAgents).toHaveBeenCalledWith(agents)
  })

  it('skips correlation while disconnected', () => {
    vi.useFakeTimers()
    subscribeBloodhound()
    requestCorrelation([{ ID: 'a1' }])
    vi.advanceTimersByTime(1000)
    expect(api.correlateAgents).not.toHaveBeenCalled()
  })
})

describe('enrichmentChip', () => {
  it('returns null without a resolved entity', () => {
    expect(enrichmentChip(null)).toBeNull()
    expect(enrichmentChip({ entity: {} })).toBeNull()
  })

  it('marks tier zero entities and distances', () => {
    expect(enrichmentChip({ entity: { objectId: 'S-1' }, tierZero: true, distanceToTierZero: 0 })).toMatchObject({
      kind: 'tierZero', label: 'T0',
    })
    expect(enrichmentChip({ entity: { objectId: 'S-1' }, distanceToTierZero: 2 })).toMatchObject({
      kind: 'tierZero', label: 'T0·2',
    })
  })

  it('marks owned entities', () => {
    expect(enrichmentChip({ entity: { objectId: 'S-1' }, owned: true, distanceToTierZero: -1 })).toMatchObject({
      kind: 'owned', label: 'OWNED',
    })
  })

  it('marks unreachable entities', () => {
    expect(enrichmentChip({ entity: { objectId: 'S-1' }, distanceToTierZero: -1 })).toMatchObject({
      kind: 'unreached', label: '—',
    })
  })

  it('builds path tooltips', () => {
    expect(pathTitle([{ label: 'jane' }, { label: 'DOMAIN ADMINS' }, { label: 'DC1' }]))
      .toBe('Path to Tier-0: jane → DOMAIN ADMINS → DC1')
    expect(pathTitle([])).toBe('No path to Tier-0 found')
  })
})
