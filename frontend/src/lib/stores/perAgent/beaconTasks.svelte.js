import { listBeaconTasks } from '../../api/agents.js'
import { createPollingPerKeyStore } from '../lib/createPerKeyStore.svelte.js'

const POLL_INTERVAL = 5000

export const useBeaconTasks = createPollingPerKeyStore((beaconID) => {
  const state = $state({
    tasks: [],
    loading: false,
    error: null,
  })

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

  return {
    state,
    refresh,
  }
}, { pollInterval: POLL_INTERVAL })
