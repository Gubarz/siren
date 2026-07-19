import { beforeEach, describe, expect, it, vi } from 'vitest'
import { installWailsApp } from '../../../../test/mocks/wails.js'
import { listEvents } from '../events.js'

describe('events api', () => {
  let app

  beforeEach(() => {
    app = installWailsApp({
      GetEventHistory: vi.fn(() => Promise.resolve([])),
    })
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
