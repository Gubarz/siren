import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const mocked = vi.hoisted(() => ({
  element: { parentElement: null },
  fit: { fit: vi.fn() },
  term: null,
  GetConsoleOutput: vi.fn(() => Promise.resolve('')),
  StartConsole: vi.fn(() => Promise.resolve('job-1')),
  StopConsole: vi.fn(() => Promise.resolve()),
  ResizeConsole: vi.fn(() => Promise.resolve()),
  WriteConsole: vi.fn(() => Promise.resolve()),
  createXterm: vi.fn(),
  onConsoleOutput: vi.fn(() => vi.fn()),
  onConsoleExit: vi.fn(() => vi.fn()),
  onConsoleOpenShell: vi.fn(() => vi.fn()),
  pageHide: null,
}))

vi.mock('../../../api/console.js', () => ({
  GetConsoleOutput: mocked.GetConsoleOutput,
  ResizeConsole: mocked.ResizeConsole,
  StartConsole: mocked.StartConsole,
  StopConsole: mocked.StopConsole,
  WriteConsole: mocked.WriteConsole,
}))

vi.mock('../../../api/runtime.js', () => ({
  onConsoleExit: mocked.onConsoleExit,
  onConsoleOpenShell: mocked.onConsoleOpenShell,
  onConsoleOutput: mocked.onConsoleOutput,
}))

vi.mock('$utils/xterm.js', () => ({
  createXterm: mocked.createXterm,
}))

describe('console PTY session cache', () => {
  let acquireConsolePty

  beforeEach(async () => {
    vi.useFakeTimers()
    vi.resetModules()
    vi.clearAllMocks()
    delete globalThis.__sliverGuiConsolePtySessionState
    mocked.pageHide = null
    vi.stubGlobal('addEventListener', vi.fn((name, callback) => {
      if (name === 'pagehide') mocked.pageHide = callback
    }))
    mocked.element.parentElement = null
    mocked.fit.fit.mockClear()
    mocked.term = {
      cols: 100,
      rows: 30,
      element: mocked.element,
      dispose: vi.fn(),
      focus: vi.fn(),
      onData: vi.fn(() => ({ dispose: vi.fn() })),
      write: vi.fn((_data, callback) => callback?.()),
    }
    mocked.createXterm.mockImplementation((host) => {
      mocked.element.parentElement = host
      return { term: mocked.term, fit: mocked.fit }
    })
    ;({ acquireConsolePty } = await import('./consolePtySession.js'))
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('keeps the same console job alive across an immediate remount', async () => {
    const firstHost = host()
    const secondHost = host()

    const first = acquireConsolePty('session-1', firstHost)
    await Promise.resolve()
    first.release()

    const second = acquireConsolePty('session-1', secondHost)
    await vi.advanceTimersByTimeAsync(3100)

    expect(mocked.StartConsole).toHaveBeenCalledTimes(1)
    expect(mocked.StopConsole).not.toHaveBeenCalled()
    expect(secondHost.appendChild).toHaveBeenCalledWith(mocked.element)

    second.release()
    await vi.advanceTimersByTimeAsync(3100)

    expect(mocked.StopConsole).toHaveBeenCalledWith('job-1')
    expect(mocked.term.dispose).toHaveBeenCalled()
  })

  it('replays buffered output when attaching to a console job', async () => {
    mocked.GetConsoleOutput.mockResolvedValueOnce(btoa('existing console output'))

    acquireConsolePty('session-1', host())
    await vi.waitFor(() => {
      expect(mocked.GetConsoleOutput).toHaveBeenCalledWith('job-1')
      expect(mocked.term.write).toHaveBeenCalled()
    })

    const replayed = mocked.term.write.mock.calls[0][0]
    expect(new TextDecoder().decode(replayed)).toBe('existing console output')
  })

  it('releases a window lease only once during page teardown', async () => {
    const handle = acquireConsolePty('session-1', host())
    await Promise.resolve()

    mocked.pageHide()
    handle.release()
    await vi.advanceTimersByTimeAsync(3100)

    expect(mocked.StopConsole).toHaveBeenCalledTimes(1)
    expect(mocked.StopConsole).toHaveBeenCalledWith('job-1')
  })

  function host() {
    const node = {
      appendChild: vi.fn((child) => {
        child.parentElement = node
      }),
    }
    return node
  }
})
