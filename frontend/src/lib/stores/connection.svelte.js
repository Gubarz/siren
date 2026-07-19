import { Connect, GetVersion } from '../api/connection.js'
import { workspaceState } from './workspaceState.svelte.js'
import { navigation } from './ui/navigation.svelte.js'

class Connection {
  connected = $state(false)
  profile = $state('')
  version = $state('')
  reconnecting = $state(false)
  error = $state(null)

  #reconnectInterval = null

  #setConnected(profile, version = '') {
    const changed = this.profile !== profile
    this.connected = true
    this.profile = profile
    this.version = version
    this.reconnecting = false
    this.error = null
    this.#stopReconnect()
    if (changed) {
      workspaceState.hydrate(profile)
      navigation.hydrateFromWorkspace()
    }
  }

  async #refreshVersion() {
    try {
      const version = await GetVersion()
      if (!this.connected) return
      this.version = `${version.Major}.${version.Minor}.${version.Patch}`
    } catch {
      // Version display is non-critical.
    }
  }

  markConnected(profile) {
    this.#setConnected(profile)
    void this.#refreshVersion()
  }

  async connect(profile) {
    try {
      await Connect(profile)
      this.markConnected(profile)
    } catch (err) {
      this.error = String(err)
      throw err
    }
  }

  disconnect() {
    this.connected = false
    this.profile = ''
    this.version = ''
    this.reconnecting = false
    this.#stopReconnect()
    workspaceState.clear()
  }

  startReconnecting(profile) {
    this.connected = false
    this.reconnecting = true
    if (!this.#reconnectInterval) {
      this.#reconnectInterval = setInterval(async () => {
        try {
          await Connect(profile)
          this.markConnected(profile)
        } catch {}
      }, 5000)
    }
  }

  updateVersion(major, minor, patch) {
    this.version = `${major}.${minor}.${patch}`
  }

  #stopReconnect() {
    if (this.#reconnectInterval) {
      clearInterval(this.#reconnectInterval)
      this.#reconnectInterval = null
    }
  }
}

export const connection = new Connection()
