const STORAGE_KEY = 'gui-command-presets'

function loadAll() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { version: 1, presets: {} }
    const data = JSON.parse(raw)
    if (data.version !== 1) return { version: 1, presets: {} }
    return data
  } catch {
    return { version: 1, presets: {} }
  }
}

function saveAll(data) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
  } catch { /* quota exceeded, silently ignore */ }
}

export const presets = {
  list(commandPath) {
    return loadAll().presets[commandPath] || []
  },

  get(commandPath, name) {
    const all = loadAll().presets[commandPath] || []
    return all.find((p) => p.name === name) || null
  },

  save(commandPath, name, values) {
    const data = loadAll()
    if (!data.presets[commandPath]) data.presets[commandPath] = []
    const existing = data.presets[commandPath].findIndex((p) => p.name === name)
    if (existing >= 0) {
      data.presets[commandPath][existing] = { name, values }
    } else {
      data.presets[commandPath].push({ name, values })
    }
    saveAll(data)
  },

  remove(commandPath, name) {
    const data = loadAll()
    if (!data.presets[commandPath]) return
    data.presets[commandPath] = data.presets[commandPath].filter((p) => p.name !== name)
    saveAll(data)
  },

  rename(commandPath, oldName, newName) {
    const data = loadAll()
    const list = data.presets[commandPath] || []
    const found = list.findIndex((p) => p.name === oldName)
    if (found >= 0) {
      list[found].name = newName
      saveAll(data)
    }
  },
}
