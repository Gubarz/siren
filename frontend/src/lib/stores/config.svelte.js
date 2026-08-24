const DEFAULT_NOTIFICATION_TYPES = {
  'session-connected': true,
  'session-disconnected': true,
  'beacon-registered': true,
  'job-started': true,
  'job-stopped': true,
}

const DEFAULTS = {
  theme: 'dark',
  agentViewMode: 'table',
  graphDirection: 'TB',
  topPaneHeight: 50,
  showEventStream: true,
  eventsHideAcked: false,
  confirmShellAdult: true,
  zoom: 1.0,
  notifications: {
    // Global mute + per-event-type toggles. Falsy = don't toast.
    enabled: true,
    types: DEFAULT_NOTIFICATION_TYPES,
    // Do-not-disturb window — 24h "HH:MM" strings. If start > end the
    // window wraps midnight (e.g. 22:00 → 08:00 mutes overnight).
    dnd: { enabled: false, start: '22:00', end: '08:00' },
  },
}

export { DEFAULT_NOTIFICATION_TYPES }

export const ZOOM_MIN = 1.0
export const ZOOM_MAX = 2.0
export const ZOOM_STEP = 0.1

export function clampZoom(z) {
  const n = Number(z)
  if (!Number.isFinite(n)) return 1.0
  return Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, Math.round(n * 100) / 100))
}

function loadPersisted() {
  try {
    const raw = localStorage.getItem('sliver-config')
    if (raw) {
      const state = { ...DEFAULTS, ...JSON.parse(raw) }
      state.zoom = clampZoom(state.zoom)
      return state
    }
  } catch {}
  return { ...DEFAULTS }
}

function persist(snapshot) {
  try {
    localStorage.setItem('sliver-config', JSON.stringify(snapshot))
  } catch {}
}

class Config {
  #state = $state(loadPersisted())

  get theme() { return this.#state.theme }
  get agentViewMode() { return this.#state.agentViewMode }
  get graphDirection() { return this.#state.graphDirection }
  get topPaneHeight() { return this.#state.topPaneHeight }
  get showEventStream() { return this.#state.showEventStream }
  get eventsHideAcked() { return this.#state.eventsHideAcked }
  get confirmShellAdult() { return this.#state.confirmShellAdult }
  get zoom() { return this.#state.zoom }
  get notifications() { return this.#state.notifications }

  set(key, value) {
    this.#state = { ...this.#state, [key]: value }
    persist(this.#state)
  }

  reset() {
    this.#state = { ...DEFAULTS }
    persist(this.#state)
  }

  setZoom(z) {
    this.set('zoom', clampZoom(z))
  }

  zoomIn() {
    this.setZoom((this.zoom ?? 1.0) + ZOOM_STEP)
  }

  zoomOut() {
    this.setZoom((this.zoom ?? 1.0) - ZOOM_STEP)
  }

  zoomReset() {
    this.setZoom(1.0)
  }
}

export const config = new Config()
