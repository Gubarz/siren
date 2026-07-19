import { listFiles, Pwd } from '../../api/agents.js'
import { createPerKeyStore } from '../lib/createPerKeyStore.svelte.js'

export const useFileBrowser = createPerKeyStore((sessionID) => {
  const state = $state({
    path: '',
    files: [],
    loading: false,
    error: null,
  })

  let hasInitialCwd = false

  async function initialCwd() {
    try {
      const path = await Pwd(sessionID)
      return path || ''
    } catch {
      return ''
    }
  }

  function refresh(path) {
    const targetPath = path ?? state.path
    state.loading = true
    state.error = null
    listFiles(sessionID, targetPath)
      .then((result) => {
        state.path = result.path
        state.files = result.files
        state.loading = false
        state.error = null
      })
      .catch((err) => {
        state.loading = false
        state.error = String(err)
      })
  }

  async function firstLoad() {
    if (hasInitialCwd) { refresh(); return }
    hasInitialCwd = true
    const cwd = await initialCwd()
    refresh(cwd)
  }

  function cd(path) {
    state.path = path
    refresh(path)
  }

  return {
    state,
    onActivate: firstLoad,
    refresh,
    cd,
  }
})
