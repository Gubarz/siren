import { beforeEach, describe, expect, it, vi } from 'vitest'
import { listExtensions, listWasmExtensions } from '../extensions.js'

const app = vi.hoisted(() => ({
  RegisterExtensionFromPath: vi.fn(),
  ListExtensions: vi.fn(),
  CallExtension: vi.fn(),
  RegisterWasmExtensionFromPath: vi.fn(),
  ListWasmExtensions: vi.fn(),
  ExecWasmExtension: vi.fn(),
}))
vi.mock('../../../../bindings/siren/cmd/gui/app.js', () => app)

describe('extensions api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('passes the session ID to extension list bindings', async () => {
    app.ListExtensions.mockResolvedValue({ Names: ['seatbelt'] })
    app.ListWasmExtensions.mockResolvedValue({ names: ['whoami'] })

    await expect(listExtensions('session-1')).resolves.toEqual(['seatbelt'])
    await expect(listWasmExtensions('session-1')).resolves.toEqual(['whoami'])

    expect(app.ListExtensions).toHaveBeenCalledWith('session-1')
    expect(app.ListWasmExtensions).toHaveBeenCalledWith('session-1')
  })

  it('falls back to empty lists when names are absent', async () => {
    app.ListExtensions.mockResolvedValue({})
    app.ListWasmExtensions.mockResolvedValue({})

    await expect(listExtensions('session-1')).resolves.toEqual([])
    await expect(listWasmExtensions('session-1')).resolves.toEqual([])
  })
})
