import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../../api/operatorControls.js', () => ({
  LogClient: vi.fn(() => Promise.resolve()),
}))

describe('installClientLogHandlers', () => {
  let LogClient
  let listeners
  let installClientLogHandlers

  beforeEach(async () => {
    vi.resetModules()
    listeners = new Map()
    globalThis.window = {
      addEventListener: vi.fn((type, callback) => listeners.set(type, callback)),
      removeEventListener: vi.fn((type, callback) => {
        if (listeners.get(type) === callback) listeners.delete(type)
      }),
    }
    ;({ LogClient } = await import('../../api/operatorControls.js'))
    ;({ installClientLogHandlers } = await import('../clientLog.js'))
    vi.clearAllMocks()
  })

  it('logs browser errors with context and source location', () => {
    const cleanup = installClientLogHandlers()

    listeners.get('error')({
      message: 'Boom',
      filename: '/src/App.svelte',
      lineno: 10,
      colno: 5,
    })

    expect(LogClient).toHaveBeenCalledWith('[window-error] Boom (/src/App.svelte:10:5)')
    cleanup()
  })

  it('logs unhandled rejections using normalized error messages', () => {
    installClientLogHandlers()

    listeners.get('unhandledrejection')({ reason: new Error('Async boom') })

    expect(LogClient).toHaveBeenCalledWith('[unhandled-rejection] Async boom')
  })

  it('removes handlers during cleanup', () => {
    const cleanup = installClientLogHandlers()

    cleanup()

    expect(window.removeEventListener).toHaveBeenCalledTimes(2)
    expect(listeners.size).toBe(0)
  })

  it('truncates long log lines before sending them to the backend', () => {
    installClientLogHandlers()

    listeners.get('unhandledrejection')({ reason: 'x'.repeat(5000) })

    const [line] = LogClient.mock.calls[0]
    expect(line).toHaveLength('[unhandled-rejection] '.length + 4000)
    expect(line).toMatch(/^\[unhandled-rejection\] x+$/)
  })
})
