import { beforeEach, describe, expect, it, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  listKnownAgents: vi.fn(),
  onSliverEvent: vi.fn(),
}))

vi.mock('../../api/agents.js', () => ({ listKnownAgents: mocks.listKnownAgents }))
vi.mock('../../api/runtime.js', () => ({ onSliverEvent: mocks.onSliverEvent }))

async function flush() {
  await Promise.resolve()
  await Promise.resolve()
}

function deferred() {
  let resolve
  const promise = new Promise((r) => { resolve = r })
  return { promise, resolve }
}

describe('knownAgents store', () => {
  let knownAgents
  let eventHandler

  beforeEach(async () => {
    vi.resetModules()
    mocks.listKnownAgents.mockReset()
    mocks.onSliverEvent.mockReset()
    mocks.onSliverEvent.mockImplementation((callback) => {
      eventHandler = callback
      return vi.fn()
    })
    eventHandler = null
    ;({ knownAgents } = await import('../knownAgents.svelte.js'))
  })

  it('loads records and normalizes null to an empty list', async () => {
    mocks.listKnownAgents.mockResolvedValue(null)

    await knownAgents.load()

    expect(knownAgents.data).toEqual([])
    expect(knownAgents.loading).toBe(false)
    expect(knownAgents.error).toBeNull()
  })

  it('derives lost records from the loaded data', async () => {
    mocks.listKnownAgents.mockResolvedValue([
      { id: 'a', status: 'active' },
      { id: 'b', status: 'lost' },
      { id: 'c', status: 'lost' },
    ])

    await knownAgents.load()

    expect(knownAgents.lost.map((r) => r.id)).toEqual(['b', 'c'])
  })

  it('refreshes on session lifecycle events and ignores unrelated ones', async () => {
    mocks.listKnownAgents.mockResolvedValue([{ id: 'a', status: 'lost' }])

    await knownAgents.load()
    expect(mocks.listKnownAgents).toHaveBeenCalledTimes(1)
    expect(typeof eventHandler).toBe('function')

    eventHandler({ type: 'session-connected' })
    await flush()
    expect(mocks.listKnownAgents).toHaveBeenCalledTimes(2)

    eventHandler({ type: 'session-closed' })
    await flush()
    expect(mocks.listKnownAgents).toHaveBeenCalledTimes(3)

    eventHandler({ type: 'session-disconnected' })
    await flush()
    expect(mocks.listKnownAgents).toHaveBeenCalledTimes(4)

    eventHandler({ type: 'job-started' })
    await flush()
    expect(mocks.listKnownAgents).toHaveBeenCalledTimes(4)
  })

  it('runs one trailing reload when an event lands mid-fetch', async () => {
    const first = deferred()
    mocks.listKnownAgents
      .mockReturnValueOnce(first.promise)
      .mockResolvedValueOnce([{ id: 'b', status: 'lost' }])

    const inFlight = knownAgents.load()
    expect(knownAgents.loading).toBe(true)

    eventHandler({ type: 'session-disconnected' })
    await flush()
    expect(mocks.listKnownAgents).toHaveBeenCalledTimes(1)

    first.resolve([{ id: 'a', status: 'active' }])
    await inFlight

    expect(mocks.listKnownAgents).toHaveBeenCalledTimes(2)
    expect(knownAgents.lost.map((r) => r.id)).toEqual(['b'])
    expect(knownAgents.loading).toBe(false)
  })

  it('clears stale rows when the sliver stream closes', async () => {
    mocks.listKnownAgents.mockResolvedValue([{ id: 'a', status: 'lost' }])

    await knownAgents.load()
    expect(knownAgents.data).toHaveLength(1)

    eventHandler({ type: 'stream-closed' })
    await flush()

    expect(knownAgents.data).toEqual([])
    expect(mocks.listKnownAgents).toHaveBeenCalledTimes(1)
  })

  it('keeps data cleared when an in-flight load resolves after stream close', async () => {
    const first = deferred()
    mocks.listKnownAgents.mockReturnValueOnce(first.promise)

    const inFlight = knownAgents.load()
    expect(knownAgents.loading).toBe(true)

    eventHandler({ type: 'stream-closed' })
    first.resolve([{ id: 'a', status: 'lost' }])
    await inFlight

    expect(knownAgents.data).toEqual([])
    expect(knownAgents.loading).toBe(false)
  })

  it('captures load errors without throwing', async () => {
    mocks.listKnownAgents.mockRejectedValue(new Error('not connected'))

    await knownAgents.load()

    expect(knownAgents.data).toEqual([])
    expect(knownAgents.error).toContain('not connected')
    expect(knownAgents.loading).toBe(false)
  })
})
