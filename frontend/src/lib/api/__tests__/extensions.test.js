import { beforeEach, describe, expect, it, vi } from 'vitest'
import { installWailsApp } from '../../../../test/mocks/wails.js'
import { listExtensions, listWasmExtensions } from '../extensions.js'

describe('extensions api', () => {
  let app

  beforeEach(() => {
    app = installWailsApp({
      ListExtensions: vi.fn(),
      ListWasmExtensions: vi.fn(),
    })
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
