import { beforeEach, describe, expect, it, vi } from 'vitest'
import { installWailsApp } from '../../../../test/mocks/wails.js'
import { listCrackFiles, listCrackstations } from '../crack.js'

describe('crack api', () => {
  let app

  beforeEach(() => {
    app = installWailsApp({
      Crackstations: vi.fn(),
      CrackFilesList: vi.fn(),
    })
  })

  it('returns crackstations from PascalCase or camelCase response fields', async () => {
    app.Crackstations.mockResolvedValueOnce({ Crackstations: [{ name: 'gpu-a' }] })
    await expect(listCrackstations()).resolves.toEqual([{ name: 'gpu-a' }])

    app.Crackstations.mockResolvedValueOnce({ crackstations: [{ name: 'gpu-b' }] })
    await expect(listCrackstations()).resolves.toEqual([{ name: 'gpu-b' }])
  })

  it('falls back to an empty crack file list', async () => {
    app.CrackFilesList.mockResolvedValue({})

    await expect(listCrackFiles()).resolves.toEqual([])
    expect(app.CrackFilesList).toHaveBeenCalledOnce()
  })

  it('surfaces binding errors as rejected promises', async () => {
    app.Crackstations.mockRejectedValue(new Error('not connected'))

    await expect(listCrackstations()).rejects.toThrow('not connected')
  })
})
