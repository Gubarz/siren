import { listBeaconTasks } from '../../api/agents.js'
import { createPerKeyStore } from '../lib/createPerKeyStore.svelte.js'

const POLL_INTERVAL = 5000

export const useBeaconTasks = createPerKeyStore((beaconID) => {
  const state = $state({
    tasks: [],
    loading: false,
    error: null,
  })

  let pollTimer = null

  async function refresh() {
    state.loading = true
    try {
      const tasks = await listBeaconTasks(beaconID)
      state.tasks = tasks
      state.error = null
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
