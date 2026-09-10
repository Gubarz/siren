// Known agents are the backend's persisted active/lost session records.
// Live sessions and beacons always win in the table; this store only feeds
// the merge that keeps lost sessions visible until removed.
import { listKnownAgents } from '../api/agents.js'
import { onSliverEvent } from '../api/runtime.js'

// The teamserver bridge emits session lifecycle events as
// session-connected/session-disconnected; opened/closed are accepted
// aliases so either naming refreshes the list.
const REFRESH_EVENT_TYPES = new Set([
  'session-connected',
  'session-disconnected',
  'session-opened',
  'session-closed',
])

class KnownAgentsStore {
  data = $state([])
  loading = $state(false)
  error = $state(null)

  #subscribed = false
  #pendingReload = false
  // Bumped by stream-closed and every load start; a stale in-flight response
  // can never repopulate state that was cleared after it began.
  #generation = 0

  get lost() {
    return this.data.filter((record) => record?.status === 'lost')
  }

  async load() {
    this.#subscribe()
    if (this.loading) {
      this.#pendingReload = true
      return
    }
    this.loading = true
    const generation = ++this.#generation
    try {
      const records = await listKnownAgents()
      if (generation === this.#generation) {
        this.data = Array.isArray(records) ? records : []
        this.error = null
      }
    } catch (err) {
      if (generation === this.#generation) {
        this.error = String(err)
      }
    } finally {
      this.loading = false
    }
    if (this.#pendingReload) {
      this.#pendingReload = false
      await this.load()
    }
  }

  // The wails subscription is process-lifetime; first load wires it and
  // later loads are no-ops, mirroring the bloodhound store pattern.
  #subscribe() {
    if (this.#subscribed) return
    this.#subscribed = true
    onSliverEvent((event) => {
      // Connection loss invalidates the whole list: rows belong to the
      // previous teamserver and must not survive a reconnect.
      if (event?.type === 'stream-closed') {
        this.#generation += 1
        this.data = []
        return
      }
      if (REFRESH_EVENT_TYPES.has(event?.type)) void this.load()
    })
  }
}

export const knownAgents = new KnownAgentsStore()
