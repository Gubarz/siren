class Selection {
  agents = $state(new Set())
  devices = $state(new Set())

  replace({ agents = [], devices = [] }) {
    this.agents = new Set(agents)
    this.devices = new Set(devices)
  }

  toggle(type, id) {
    const key = type === 'agent' ? 'agents' : 'devices'
    const next = new Set(this[key])
    if (next.has(id)) next.delete(id)
    else next.add(id)
    this[key] = next
  }

  select(type, id, additive = false) {
    if (additive) {
      this.toggle(type, id)
      return
    }
    const key = type === 'agent' ? 'agents' : 'devices'
    this.agents = new Set()
    this.devices = new Set()
    this[key] = new Set([id])
  }

  clear() {
    this.agents = new Set()
    this.devices = new Set()
  }
}

export const selection = new Selection()
