import { listProcesses } from '../../api/agents.js'
import { createPerKeyStore } from '../lib/createPerKeyStore.svelte.js'

export const useProcessList = createPerKeyStore((sessionID) => {
  const state = $state({
    processes: [],
    loading: false,
    error: null,
    isTreeView: false,
    isFullView: false,
  })

  function refresh(fullView) {
    const fv = fullView !== undefined ? fullView : state.isFullView
    state.loading = true
    state.error = null
    state.isFullView = fv
    listProcesses(sessionID, fv)
      .then((processes) => {
        state.processes = processes
        state.loading = false
        state.error = null
      })
      .catch((err) => {
        state.loading = false
        state.error = String(err)
      })
  }

  function setTreeView(val) {
    state.isTreeView = val
  }

  function setFullView(val) {
    refresh(val)
  }

  return {
    state,
    onActivate: () => refresh(state.isFullView),
    refresh,
    setTreeView,
    setFullView,
  }
})
