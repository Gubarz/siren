import { beforeEach, describe, expect, it, vi } from 'vitest'

const app = vi.hoisted(() => ({
  UnstageImplantBuild: vi.fn(),
  UnstageAllImplantBuilds: vi.fn(),
}))
vi.mock('../../../../bindings/siren/cmd/gui/app.js', () => app)

import { UnstageAllImplantBuilds, UnstageImplantBuild } from '../staging.js'

describe('staging api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('re-exports unstage bindings', async () => {
    await UnstageImplantBuild('alpha')
    expect(app.UnstageImplantBuild).toHaveBeenCalledWith('alpha')

    await UnstageAllImplantBuilds()
    expect(app.UnstageAllImplantBuilds).toHaveBeenCalledOnce()
  })
})
