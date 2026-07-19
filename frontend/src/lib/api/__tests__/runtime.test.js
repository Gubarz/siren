import { beforeEach, describe, expect, it, vi } from 'vitest'
import { installWailsRuntime } from '../../../../test/mocks/wails.js'

describe('runtime api', () => {
  let runtime
  let api

  beforeEach(async () => {
    vi.resetModules()
    runtime = installWailsRuntime({
      EventsOnMultiple: vi.fn(() => vi.fn()),
      OnFileDrop: vi.fn(),
      OnFileDropOff: vi.fn(),
      WindowMinimise: vi.fn(),
      WindowToggleMaximise: vi.fn(),
      Quit: vi.fn(),
    })
    api = await import('../runtime.js')
  })

  it('subscribes to Sliver events on the canonical Wails channel', () => {
    const callback = vi.fn()
    const unsubscribe = vi.fn()
    runtime.EventsOnMultiple.mockReturnValue(unsubscribe)

    expect(api.onSliverEvent(callback)).toBe(unsubscribe)
    expect(runtime.EventsOnMultiple).toHaveBeenCalledWith('sliver-event', callback, -1)
  })

  it('multiplexes file-drop listeners onto one Wails registration', () => {
    const first = vi.fn()
    const second = vi.fn()

    const offFirst = api.onFileDrop(first)
    const offSecond = api.onFileDrop(second)
    const [handler, includeDirectories] = runtime.OnFileDrop.mock.calls[0]
    handler(12, 34, ['/tmp/payload.bin'])

    expect(runtime.OnFileDrop).toHaveBeenCalledTimes(1)
    expect(includeDirectories).toBe(true)
    expect(first).toHaveBeenCalledWith(12, 34, ['/tmp/payload.bin'])
    expect(second).toHaveBeenCalledWith(12, 34, ['/tmp/payload.bin'])

    offFirst()
    expect(runtime.OnFileDropOff).not.toHaveBeenCalled()
    offSecond()
    expect(runtime.OnFileDropOff).toHaveBeenCalledOnce()
  })
})
