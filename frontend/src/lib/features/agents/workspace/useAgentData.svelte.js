// Pure data engine for AgentTopPane. Takes plain arrays and the current
// filter needle, returns every derived list the component needs. Zero
// store subscriptions here — the caller owns those and hands us the
// current snapshot each time it recomputes.
//
// Kept as a factory so future variants (saved views, per-workspace
// filtering) can plug in without touching the call site.

import { agentMatchesNeedle, deviceMatchesNeedle } from './agentFilter.js'
import { dedupeDiscoveries, pivotParentMap } from '../../../utils/agents.js'

export function createAgentDataModel() {
  function process(agentFilter, sessionData, beaconData, rawDiscoveredData, pivotData, tagsByAgent, knownAgents) {
    const needle = ((agentFilter ?? '') + '').trim().toLowerCase()
    const discoveredData = dedupeDiscoveries(rawDiscoveredData || [])
    const combinedData = [
      ...(beaconData || []).map((b) => ({ ...b, _kind: 'beacon' })),
      ...(sessionData || []).map((s) => ({ ...s, _kind: 'session' })),
    ]
    const lostData = mergeLostAgents(combinedData, knownAgents)
    const allData = [...combinedData, ...lostData]

    const filteredData = !needle
      ? allData
      : allData.filter((agent) => agentPassesFilter(agent, needle, tagsByAgent, discoveredData))

    const filteredDiscoveries = !needle
      ? discoveredData
      : discoveredData.filter((d) => devicePassesFilter(d, needle, combinedData, tagsByAgent))

    const graphAgentIDs = buildGraphIDSet(
      needle,
      combinedData,
      filteredData.filter((agent) => !agent._lost),
      pivotData,
    )
    const graphSessions = (sessionData || []).filter((s) => graphAgentIDs.has(s.ID))
    const graphBeacons = (beaconData || []).filter((b) => graphAgentIDs.has(b.ID))

    return {
      needle,
      combinedData: allData,
      discoveredData,
      filteredData,
      filteredDiscoveries,
      graphSessions,
      graphBeacons,
    }
  }

  return { process }
}

// Lost records whose IDs are still live are dropped — a session that came
// back must not render twice. Ordering keeps newest last-seen first, after
// every live entry.
export function mergeLostAgents(liveData, knownAgents) {
  const liveIDs = new Set(liveData.map((agent) => agent.ID))
  return (knownAgents || [])
    .filter((record) => record?.status === 'lost')
    .filter((record) => !liveIDs.has(record.ID ?? record.id))
    .sort((a, b) => Number(b.lastSeen ?? 0) - Number(a.lastSeen ?? 0))
    .map(lostAgent)
}

// The bindings deliver camelCase records; the table and filter helpers read
// the live (PascalCase) shape, so alias the identity fields onto the record.
// Aliases only fill gaps, so an already-normalized record is returned as-is.
function lostAgent(record) {
  const merged = {
    ...record,
    _kind: record.kind ?? record._kind ?? 'session',
    _lost: true,
  }
  aliasField(merged, 'ID', record.id)
  aliasField(merged, 'Name', record.name)
  aliasField(merged, 'Hostname', record.hostname)
  aliasField(merged, 'Username', record.username)
  aliasField(merged, 'OS', record.os)
  aliasField(merged, 'Arch', record.arch)
  aliasField(merged, 'RemoteAddress', record.remoteAddress)
  aliasField(merged, 'Transport', record.transport)
  return merged
}

function aliasField(target, key, value) {
  if (target[key] == null && value != null) target[key] = value
}

// agentPassesFilter — an agent matches its own fields OR any device it
// observed matches the needle. Second clause pulls the parent agent in
// when an operator searches for a discovered host/IP.
function agentPassesFilter(agent, needle, tagsByAgent, discoveredData) {
  if (agentMatchesNeedle(agent, needle, tagsByAgent)) return true
  for (const d of discoveredData) {
    if (!deviceMatchesNeedle(d, needle)) continue
    const observerIDs = d.observerIDs || [d.agentID]
    if (observerIDs.includes(agent.ID)) return true
  }
  return false
}

// devicePassesFilter — a device matches its own fields OR any of its
// observer agents match. Second clause keeps a device visible when the
// operator filtered by hostname/user and a matching agent observed it.
function devicePassesFilter(device, needle, combinedData, tagsByAgent) {
  if (deviceMatchesNeedle(device, needle)) return true
  const observerIDs = device.observerIDs || [device.agentID]
  return observerIDs.some((id) => {
    const agent = combinedData.find((a) => a.ID === id)
    return agent && agentMatchesNeedle(agent, needle, tagsByAgent)
  })
}

// buildGraphIDSet walks each matching agent's pivot chain up to the
// root so ancestor listeners stay visible even when they don't match
// the needle themselves — otherwise a pivot subtree renders orphaned.
function buildGraphIDSet(needle, combinedData, filteredData, pivotData) {
  if (!needle) return new Set(combinedData.map((a) => a.ID))
  const parents = pivotParentMap(pivotData)
  const visible = new Set(filteredData.map((a) => a.ID))
  for (const agent of filteredData) {
    let parentID = parents.get(agent.ID)
    while (parentID && !visible.has(parentID)) {
      visible.add(parentID)
      parentID = parents.get(parentID)
    }
  }
  return visible
}
