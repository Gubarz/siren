import { beforeEach, describe, expect, it, vi } from 'vitest'
import { listImplantBuilds } from '../server.js'

const app = vi.hoisted(() => ({
  GetImplantBuilds: vi.fn(),
}))
vi.mock('../../../../bindings/siren/cmd/gui/app.js', () => app)

describe('listImplantBuilds', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('maps builds with staged flag and nonce from ResourceIDs', async () => {
    app.GetImplantBuilds.mockResolvedValue({
      Configs: {
        alpha: { GOOS: 'linux', GOARCH: 'amd64' },
        beta: { GOOS: 'windows', GOARCH: 'amd64' },
      },
      Staged: { alpha: true },
      ResourceIDs: {
        alpha: { Value: 123456789 },
        beta: { value: 987654321 },
      },
    })

    const builds = await listImplantBuilds()

    expect(builds).toHaveLength(2)
    expect(builds[0]).toMatchObject({ name: 'alpha', staged: true, nonce: 123456789 })
    expect(builds[1]).toMatchObject({ name: 'beta', staged: false, nonce: 987654321 })
  })

  it('defaults nonce to null when ResourceIDs are missing', async () => {
    app.GetImplantBuilds.mockResolvedValue({ Configs: { alpha: {} }, Staged: {} })

    const builds = await listImplantBuilds()

    expect(builds[0]).toMatchObject({ name: 'alpha', staged: false, nonce: null })
  })

  it('falls back to empty configs for empty responses', async () => {
    app.GetImplantBuilds.mockResolvedValue({})

    await expect(listImplantBuilds()).resolves.toEqual([])
  })
})
