import { listWGSocks, listWGForwarders } from '../../api/wireguard.js'
import { createPerKeyStore } from '../lib/createPerKeyStore.svelte.js'

const POLL_INTERVAL = 5000

// WireGuard tunnels are per-session — one refresh cadence per agent, torn
// down when the last consumer for that session releases.
export const useWGTunnels = createPerKeyStore((sessionID) => {
  const state = $state({
    socksServers: [],
    forwarders: [],
    loading: false,
    error: null,
  })

  let pollTimer = null

  async function refresh() {
    state.loading = true
    state.error = null
    try {
      const [socksServers, forwarders] = await Promise.all([
        listWGSocks(sessionID),
        listWGForwarders(sessionID),
      ])
      state.socksServers = socksServers
      state.forwarders = forwarders
    } catch (err) {
      state.error = String(err)
    } finally {
      state.loading = false
    }
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  function startPolling() {
    stopPolling()
    refresh()
    pollTimer = setInterval(refresh, POLL_INTERVAL)
  }

  return {
    state,
    onActivate: startPolling,
    onDeactivate: stopPolling,
    refresh,
  }
})
