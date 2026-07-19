// Search-string matchers for the AgentTopPane filter box. Pure functions,
// no store reads.


const AGENT_FIELDS = [
  'ID', 'Name', 'Hostname', 'Username', 'OS', 'Transport',
  'RemoteAddress', 'Filename', 'ProcessName', 'PID', '_kind',
]

export function agentMatchesNeedle(agent, needle, tagsByAgent = {}) {
  for (const field of AGENT_FIELDS) {
    if (String(agent[field] ?? '').toLowerCase().includes(needle)) return true
  }
  if (String(agent.Online ? 'online' : 'offline').includes(needle)) return true
  // Tag matching — supports "#tag" and plain "tag" queries.
  const tags = tagsByAgent[agent.ID]
  if (tags && tags.length > 0) {
    const q = needle.startsWith('#') ? needle.slice(1) : needle
    if (tags.some((t) => t.includes(q))) return true
  }
  return false
}

const DEVICE_FIELDS = ['ip', 'mac', 'hostname', 'vendor', 'osHint', 'method']

export function deviceMatchesNeedle(device, needle) {
  for (const field of DEVICE_FIELDS) {
    if (String(device[field] ?? '').toLowerCase().includes(needle)) return true
  }
  return 'device discovered'.includes(needle)
}


