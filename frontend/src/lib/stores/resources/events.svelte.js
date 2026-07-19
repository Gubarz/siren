import { listEvents } from '../../api/events.js'

const INITIAL_EVENT_LIMIT = 300
const EVENT_PAGE_SIZE = 300
const MAX_EVENT_LIMIT = 2000

function normalizeEvent(event = {}) {
  return {
    ...event,
    type: event.type || event.Type || '',
    data: event.data || event.Data || '',
    sessionID: event.sessionID || event.SessionID || '',
    hostname: event.hostname || event.Hostname || '',
    username: event.username || event.Username || '',
    job: event.job || event.Job || '',
    time: event.time || event.Time || Date.now(),
  }
}

class EventLog {
  events = $state([])
  limit = $state(INITIAL_EVENT_LIMIT)
  loading = $state(false)

  #hydrated = false
  #refCount = 0

  get hasMore() {
    return this.events.length >= this.limit && this.limit < MAX_EVENT_LIMIT
  }

  async refresh() {
    if (this.loading) return
    this.loading = true
    try {
      const events = await listEvents({ limit: this.limit })
      this.events = (events || []).map(normalizeEvent).slice(-this.limit)
      this.#hydrated = true
    } finally {
      this.loading = false
    }
  }

  async loadMore() {
    if (!this.hasMore || this.loading) return
    this.limit = Math.min(MAX_EVENT_LIMIT, this.limit + EVENT_PAGE_SIZE)
    await this.refresh()
  }

  push(event) {
    const next = normalizeEvent(event)
    this.events = [...this.events.slice(-(this.limit - 1)), next]
  }

  clear() {
    this.events = []
  }

  acquire() {
    this.#refCount++
    if (!this.#hydrated) {
      Promise.resolve().then(() => {
        if (this.#refCount > 0 && !this.#hydrated) this.refresh()
      })
    }
    return this
  }

  release() {
    if (this.#refCount > 0) this.#refCount--
  }
}

export const eventLog = new EventLog()

export function pushEvent(event) {
  eventLog.push({ ...event, time: Date.now() })
}

export function clearEvents() {
  eventLog.clear()
}
