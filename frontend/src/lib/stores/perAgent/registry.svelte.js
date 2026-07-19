import { ListRegistrySubKeys, ListRegistryValues, ReadRegistryValue } from '../../api/agents.js'
import { createPerKeyStore } from '../lib/createPerKeyStore.svelte.js'

export const useRegistry = createPerKeyStore((sessionID) => {
  const state = $state({
    path: '',
    hive: 'HKLM',
    keys: [],
    values: [],
    loading: false,
    error: null,
  })

  async function refresh(hive, path) {
    const targetPath = path ?? ''
    const targetHive = hive ?? 'HKLM'
    state.loading = true
    state.error = null
    try {
      const [keysResponse, valuesResponse] = await Promise.all([
        ListRegistrySubKeys(sessionID, targetHive, targetPath),
        ListRegistryValues(sessionID, targetHive, targetPath),
      ])
      const rawKeys = keysResponse.Subkeys || keysResponse.SubKeys || []
      const valueNames = valuesResponse.ValueNames || valuesResponse.Values || []
      const values = await Promise.all(valueNames.map(async (name) => {
        try {
          const value = await ReadRegistryValue(sessionID, targetHive, targetPath, name)
          return {
            name,
            type: value.type || value.Type || 'Value',
            value: value.value ?? value.Value ?? '',
          }
        } catch {
          return { name, type: 'Value', value: '<unavailable>' }
        }
      }))
      state.path = targetPath
      state.hive = targetHive
      state.keys = rawKeys
      state.values = values
      state.loading = false
      state.error = null
    } catch (err) {
      state.loading = false
      state.error = String(err)
    }
  }

  return {
    state,
    onActivate: () => refresh(),
    refresh,
    navigate: (hive, path) => refresh(hive, path),
  }
})
