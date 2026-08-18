import { listWGSocks, listWGForwarders } from '../../api/wireguard.js'
import { createPollingPerKeyStore } from '../lib/createPerKeyStore.svelte.js'

const POLL_INTERVAL = 5000

// WireGuard tunnels are per-session — one refresh cadence per agent, torn
// down when the last consumer for that session releases.
export const useWGTunnels = createPollingPerKeyStore((sessionID) => {
  const state = $state({
    socksServers: [],
    forwarders: [],
    loading: false,
    error: null,
  })

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

  return {
    state,
    refresh,
  }
}, { pollInterval: POLL_INTERVAL })
