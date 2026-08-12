import { beforeEach, describe, expect, it, vi } from 'vitest'
import { listEvents } from '../events.js'

// The generated bindings are gitignored build output, and v3 bindings call
// through Call.ByID rather than v2's window.go globals — mock the module.
const app = vi.hoisted(() => ({
  GetEventHistory: vi.fn(),
  SetEventsAcknowledged: vi.fn(),
}))
vi.mock('../../../../bindings/siren/cmd/gui/app.js', () => app)

describe('events api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    app.GetEventHistory.mockResolvedValue([])
  })

  it('uses stable defaults for event history queries', async () => {
    await expect(listEvents()).resolves.toEqual([])

    expect(app.GetEventHistory).toHaveBeenCalledWith(0, 300)
  })

  it('passes custom since and limit values to the binding', async () => {
    app.GetEventHistory.mockResolvedValue([{ type: 'client-joined' }])

    await expect(listEvents({ since: 1234, limit: 25 })).resolves.toEqual([
      { type: 'client-joined' },
    ])
    expect(app.GetEventHistory).toHaveBeenCalledWith(1234, 25)
  })
})
