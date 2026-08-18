import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createPollingPerKeyStore } from '../createPerKeyStore.svelte.js'

describe('createPollingPerKeyStore', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('starts polling when acquired and refreshes on interval', () => {
    const refresh = vi.fn()
    const useStore = createPollingPerKeyStore((id) => ({
      state: { id },
      refresh,
    }), { pollInterval: 1000, timeout: 500 })

    const handle = useStore('key-1')
    expect(refresh).not.toHaveBeenCalled()

    handle.acquire()
    expect(refresh).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(1000)
    expect(refresh).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(2000)
    expect(refresh).toHaveBeenCalledTimes(4)

    // Release stops polling after timeout
    handle.release()
    vi.advanceTimersByTime(500)
    vi.advanceTimersByTime(2000)
    expect(refresh).toHaveBeenCalledTimes(4)
  })
})
