import { beforeEach, describe, expect, it, vi } from 'vitest'

const app = vi.hoisted(() => ({
  OpenFileDialog: vi.fn(),
}))
const runtime = vi.hoisted(() => ({
  Events: { On: vi.fn(() => vi.fn()) },
  Window: { Close: vi.fn(), Minimise: vi.fn(), Name: vi.fn(() => Promise.resolve('main')), ToggleMaximise: vi.fn() },
  Application: { Quit: vi.fn() },
}))
vi.mock('../../../../bindings/siren/cmd/gui/app.js', () => app)
vi.mock('@wailsio/runtime', () => runtime)

describe('runtime api', () => {
  let api

  beforeEach(async () => {
    vi.resetModules()
    vi.clearAllMocks()
    runtime.Events.On.mockImplementation(() => vi.fn())
    api = await import('../runtime.js')
  })

  it('subscribes to Sliver events on the canonical Wails channel', () => {
    const callback = vi.fn()
    const unsubscribe = vi.fn()
    runtime.Events.On.mockReturnValueOnce(unsubscribe)

    expect(api.onSliverEvent(callback)).toBe(unsubscribe)
    expect(runtime.Events.On).toHaveBeenCalledWith('sliver-event', expect.any(Function))
  })

  it('unwraps the v3 event envelope before invoking callbacks', () => {
    const callback = vi.fn()
    api.onSliverEvent(callback)

    const [, handler] = runtime.Events.On.mock.calls[0]
    handler({ data: { type: 'session-opened' } })

    expect(callback).toHaveBeenCalledWith({ type: 'session-opened' })
  })

  it('multiplexes file-drop listeners onto one window-scoped Wails subscription', async () => {
    const first = vi.fn()
    const second = vi.fn()

    const offFirst = api.onFileDrop(first)
    api.onFileDrop(second)
    const [name, handler] = runtime.Events.On.mock.calls[0]
    expect(name).toBe('files-dropped')
    handler({ data: { x: 12, y: 34, files: ['/tmp/payload.bin'] } })

    expect(runtime.Events.On).toHaveBeenCalledTimes(1)
    expect(first).toHaveBeenCalledWith(12, 34, ['/tmp/payload.bin'])
    expect(second).toHaveBeenCalledWith(12, 34, ['/tmp/payload.bin'])

    offFirst()
    handler({ data: { x: 1, y: 2, files: [] } })
    expect(first).toHaveBeenCalledTimes(1)
    expect(second).toHaveBeenCalledTimes(2)

    await Promise.resolve()
    handler({ sender: 'agent-tab-other', data: { x: 1, y: 2, files: ['/tmp/other.bin'] } })
    expect(second).toHaveBeenCalledTimes(2)
  })

  it('routes window controls through the v3 runtime', () => {
    api.minimizeWindow()
    api.toggleMaximizeWindow()
    api.closeWindow()
    api.quitApplication()

    expect(runtime.Window.Minimise).toHaveBeenCalledOnce()
    expect(runtime.Window.ToggleMaximise).toHaveBeenCalledOnce()
    expect(runtime.Window.Close).toHaveBeenCalledOnce()
    expect(runtime.Application.Quit).toHaveBeenCalledOnce()
  })
})
