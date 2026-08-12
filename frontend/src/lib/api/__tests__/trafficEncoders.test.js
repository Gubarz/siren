import { beforeEach, describe, expect, it, vi } from 'vitest'
import { listTrafficEncoders } from '../trafficEncoders.js'

const app = vi.hoisted(() => ({
  AddTrafficEncoder: vi.fn(),
  GetTrafficEncoderMap: vi.fn(),
  RemoveTrafficEncoder: vi.fn(),
}))
vi.mock('../../../../bindings/siren/cmd/gui/app.js', () => app)

describe('trafficEncoders api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('normalizes and sorts traffic encoders from the binding map', async () => {
    app.GetTrafficEncoderMap.mockResolvedValue({
      Encoders: {
        zeta: { ID: 7, Wasm: { Name: 'zeta', Data: [1, 2, 3] } },
        alpha: { id: 2, wasm: { data: [1] } },
      },
    })

    await expect(listTrafficEncoders()).resolves.toMatchObject([
      { id: 2, name: 'alpha', size: 1 },
      { id: 7, name: 'zeta', size: 3 },
    ])
  })

  it('falls back to an empty list when no encoders are returned', async () => {
    app.GetTrafficEncoderMap.mockResolvedValue({})

    await expect(listTrafficEncoders()).resolves.toEqual([])
  })
})
