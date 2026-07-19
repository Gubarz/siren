import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../../api/events.js', () => ({
  listEvents: vi.fn(),
}))

describe('eventLog store', () => {
  let eventLog
  let pushEvent
  let clearEvents
  let listEvents

  beforeEach(async () => {
    vi.resetModules()
    ;({ listEvents } = await import('../../../api/events.js'))
    ;({ eventLog, pushEvent, clearEvents } = await import('../events.svelte.js'))
    vi.clearAllMocks()
  })

  it('loads and normalizes the latest events within the current limit', async () => {
    listEvents.mockResolvedValue([
      { Type: 'old', Data: '1', Time: 1 },
      { Type: 'newer', SessionID: 'abc', Hostname: 'host', Username: 'user', Job: 'job', Time: 2 },
      { type: 'newest', data: '3', time: 3 },
    ])
    eventLog.limit = 2

    await eventLog.refresh()

    expect(listEvents).toHaveBeenCalledWith({ limit: 2 })
    expect(eventLog.events).toEqual([
      expect.objectContaining({
        type: 'newer',
        sessionID: 'abc',
        hostname: 'host',
        username: 'user',
        job: 'job',
        time: 2,
      }),
      expect.objectContaining({ type: 'newest', data: '3', time: 3 }),
    ])
    expect(eventLog.loading).toBe(false)
  })

  it('does not start a second refresh while one is already loading', async () => {
    let resolveEvents
    listEvents.mockReturnValue(new Promise((resolve) => {
      resolveEvents = resolve
    }))

    const first = eventLog.refresh()
    const second = eventLog.refresh()

    expect(listEvents).toHaveBeenCalledOnce()

    resolveEvents([{ type: 'ready', time: 1 }])
    await Promise.all([first, second])

    expect(eventLog.events).toEqual([expect.objectContaining({ type: 'ready' })])
    expect(eventLog.loading).toBe(false)
  })

  it('loads more events in bounded pages', async () => {
    listEvents.mockResolvedValue([])
    eventLog.events = Array.from({ length: 300 }, (_, index) => ({ type: 'event', time: index }))

    await eventLog.loadMore()

    expect(eventLog.limit).toBe(600)
    expect(listEvents).toHaveBeenCalledWith({ limit: 600 })
  })

  it('pushes timestamped events and trims the oldest entries', () => {
    vi.spyOn(Date, 'now').mockReturnValue(1234)
    eventLog.limit = 3
    eventLog.events = [
      { type: 'one', time: 1 },
      { type: 'two', time: 2 },
      { type: 'three', time: 3 },
    ]

    pushEvent({ Type: 'four', Data: 'payload' })

    expect(eventLog.events).toEqual([
      { type: 'two', time: 2 },
      { type: 'three', time: 3 },
      expect.objectContaining({ type: 'four', data: 'payload', time: 1234 }),
    ])
  })

  it('clears event history without changing the page limit', () => {
    eventLog.limit = 600
    eventLog.events = [{ type: 'event', time: 1 }]

    clearEvents()

    expect(eventLog.events).toEqual([])
    expect(eventLog.limit).toBe(600)
  })
})
