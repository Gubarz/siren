const DEFAULT_TIMEOUT = 5 * 60 * 1000
const DEFAULT_POLL_INTERVAL = 5000

// Registry of key -> per-instance handle. Each handle exposes acquire/
// release (refcounted) plus any methods returned by the create function,
// and reads the reactive state directly from the object create() built.
export function createPerKeyStore(create, { timeout = DEFAULT_TIMEOUT } = {}) {
  const instances = new Map()

  return function useInstance(key) {
    const existing = instances.get(key)
    if (existing) return existing

    const built = create(key)
    const { onActivate, onDeactivate, ...rest } = built

    let refCount = 0
    let timeoutId = null

    function acquire() {
      refCount++
      if (refCount === 1) {
        if (timeoutId) {
          clearTimeout(timeoutId)
          timeoutId = null
        }
        onActivate?.()
      }
    }

    function release() {
      if (refCount <= 0) return
      refCount--
      if (refCount === 0) {
        if (timeoutId) clearTimeout(timeoutId)
        timeoutId = setTimeout(() => {
          onDeactivate?.()
          instances.delete(key)
        }, timeout)
      }
    }

    const handle = { ...rest, acquire, release }
    instances.set(key, handle)
    return handle
  }
}

// Higher-order per-key store for resources requiring periodic polling while
// active consumers hold an acquire lease.
export function createPollingPerKeyStore(create, { pollInterval = DEFAULT_POLL_INTERVAL, timeout = DEFAULT_TIMEOUT } = {}) {
  return createPerKeyStore((key) => {
    const built = create(key)
    let pollTimer = null

    function stopPolling() {
      if (pollTimer) {
        clearInterval(pollTimer)
        pollTimer = null
      }
      built.onDeactivate?.()
    }

    function startPolling() {
      stopPolling()
      built.refresh?.()
      pollTimer = setInterval(() => built.refresh?.(), pollInterval)
      built.onActivate?.()
    }

    return {
      ...built,
      onActivate: startPolling,
      onDeactivate: stopPolling,
    }
  }, { timeout })
}
