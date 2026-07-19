// Per-teamserver UI state: last active view, server sub-tab, agent filter,
// and split ratio. Hydrated on connect from localStorage; each setter
// persists immediately. Keyed by connection profile so "reopen where I
// left off" holds per server.

const KEY_PREFIX = 'sliver-workspace:'

const DEFAULTS = {
  activeView: 'agents',
  serverTab: 'listeners',
  agentFilter: '',
  topPaneHeight: null,
}

function storageKey(profile) {
  return profile ? KEY_PREFIX + profile : ''
}

function loadFor(profile) {
  if (!profile) return { ...DEFAULTS }
  try {
    const raw = localStorage.getItem(storageKey(profile))
    if (raw) return { ...DEFAULTS, ...JSON.parse(raw) }
  } catch {}
  return { ...DEFAULTS }
}

class WorkspaceState {
  profile = $state('')
  #state = $state({ ...DEFAULTS })

  get activeView() { return this.#state.activeView }
  get serverTab() { return this.#state.serverTab }
  get agentFilter() { return this.#state.agentFilter }
  get topPaneHeight() { return this.#state.topPaneHeight }

  hydrate(profile) {
    this.profile = profile || ''
    this.#state = loadFor(this.profile)
  }

  clear() {
    this.profile = ''
    this.#state = { ...DEFAULTS }
  }

  set(key, value) {
    if (!(key in DEFAULTS)) return
    if (this.#state[key] === value) return
    this.#state = { ...this.#state, [key]: value }
    this.#persist()
  }

  #persist() {
    const key = storageKey(this.profile)
    if (!key) return
    try {
      localStorage.setItem(key, JSON.stringify(this.#state))
    } catch {}
  }
}

export const workspaceState = new WorkspaceState()
