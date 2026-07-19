import { onSliverEvent, onWailsEvent } from '../../api/runtime.js'

class Resource {
  data = $state([])
  loading = $state(false)
  error = $state(null)
  lastFetched = $state(0)

  #config
  #fetching = false
  #retryUnfetchedAfterFetch = false
  #pollTimer = null
  #unsubscribeEvents = null
  #refCount = 0

  constructor(config) {
    this.#config = config
  }

  async refresh() {
    if (this.#fetching) return
    this.#fetching = true
    this.loading = true
    this.error = null
    try {
      const result = await this.#config.fetch()
      this.data = result
      this.lastFetched = Date.now()
      this.error = null
    } catch (err) {
      this.error = String(err)
    } finally {
      this.loading = false
      this.#fetching = false
      if (this.#retryUnfetchedAfterFetch && this.#refCount > 0 && this.lastFetched === 0) {
        this.#retryUnfetchedAfterFetch = false
        Promise.resolve().then(() => this.refresh())
      } else {
        this.#retryUnfetchedAfterFetch = false
      }
    }
  }

  #refreshIfUnfetched() {
    Promise.resolve().then(() => {
      if (this.#refCount <= 0 || this.lastFetched !== 0) return
      if (this.#fetching) {
        this.#retryUnfetchedAfterFetch = true
        return
      }
      this.refresh()
    })
  }

  acquire() {
    this.#refCount++
    if (this.#refCount === 1) {
      const events = this.#config.events ?? []
      if (events.length > 0) {
        // Subscribe to BOTH channels: sliver's teamserver events (routed
        // through the sliver-event wails channel with a .type payload) and
        // any custom wails event of the same name emitted directly by the
        // Go side. The fetching guard in refresh() collapses back-to-back
        // refreshes so double-hits are harmless.
        const unsubSliver = onSliverEvent((event) => {
          if (events.includes(event.type)) this.refresh()
        })
        const customUnsubs = events
          .map((name) => onWailsEvent(name, () => this.refresh()))
          .filter((u) => typeof u === 'function')
        this.#unsubscribeEvents = () => {
          unsubSliver()
          for (const unsub of customUnsubs) unsub()
        }
      }
      if (this.#config.pollInterval > 0) {
        this.startPolling()
      } else {
        // Defer so the first read sees the initial empty state.
        this.#refreshIfUnfetched()
      }
    } else if (!(this.#config.pollInterval > 0)) {
      this.#refreshIfUnfetched()
    }
    return this
  }

  release() {
    if (this.#refCount <= 0) return
    this.#refCount--
    if (this.#refCount === 0) {
      this.stopPolling()
      if (this.#unsubscribeEvents) {
        this.#unsubscribeEvents()
        this.#unsubscribeEvents = null
      }
    }
  }

  startPolling(ms) {
    this.stopPolling()
    const interval = ms || this.#config.pollInterval
    if (interval > 0) {
      this.refresh()
      this.#pollTimer = setInterval(() => this.refresh(), interval)
    }
  }

  stopPolling() {
    if (this.#pollTimer) {
      clearInterval(this.#pollTimer)
      this.#pollTimer = null
    }
  }
}

export function createResource(config) {
  return new Resource(config)
}

// Component-side helper: acquires on mount, releases on destroy. Multiple
// components can safely share a resource — internal refcount collapses
// redundant fetches and only tears down event/poll subscriptions after
// the last consumer unmounts.
export function useResource(...resources) {
  $effect(() => {
    for (const r of resources) r.acquire()
    return () => {
      for (const r of resources) r.release()
    }
  })
}
