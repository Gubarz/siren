// Node/edge builders for NetworkGraph. Every function takes an explicit
// ctx bag so nothing here reaches into component state.

import {
  agentRemoteAddress,
  isAgentOnline,
  isHighPrivilege,
  shortAgentID,
} from '../../utils/agents.js'
import {
  c2Details,
  pivotParentFromC2,
  endpointHost,
  endpointPort,
} from './layout.js'

export const SERVER_W = 224
export const SERVER_H = 44
const LISTENER_W = 288
const LISTENER_H = 40
const NODE_W = 256
const NODE_H = 144
const DEVICE_W = 256
const DEVICE_H = 144

export function collectAgents({ sessions, beacons }) {
  return [
    ...(sessions || []).map((agent) => ({ ...agent, _kind: 'session' })),
    ...(beacons || []).map((agent) => ({ ...agent, _kind: 'beacon' })),
  ]
}

export function indexAgents(allAgents) {
  const allAgentIds = new Set(allAgents.map((agent) => agent.ID))
  const agentsByName = new Map()
  const agentsByAddress = new Map()
  for (const agent of allAgents) {
    if (agent.RemoteAddress) {
      agentsByAddress.set(String(agent.RemoteAddress).toLowerCase(), agent.ID)
    }
    for (const name of [agent.Name, agent.Hostname]) {
      if (name && !agentsByName.has(name.toLowerCase())) {
        agentsByName.set(name.toLowerCase(), agent.ID)
      }
    }
  }
  return { allAgentIds, agentsByName, agentsByAddress }
}

function osIcon(os) {
  const value = (os || '').toLowerCase()
  if (value.includes('win')) return 'monitor'
  if (value.includes('darwin') || value.includes('mac')) return 'apple'
  if (value.includes('linux')) return 'terminal'
  return 'cpu'
}

export function agentNode(impl, ctx) {
  const { parentBySession, allAgents, now, direction, selectedAgentIDs, colorsByAgent, tagsByAgent } = ctx
  return {
    id: impl.ID, w: NODE_W, h: NODE_H,
    data: {
      variant: 'agent', kind: impl._kind, icon: osIcon(impl.OS),
      entityID: impl.ID,
      agentID: shortAgentID(impl.ID),
      implantName: impl.Name || '-',
      user: impl.Username || '?', host: impl.Hostname || '?',
      addr: agentRemoteAddress(impl, parentBySession, allAgents),
      dead: !isAgentOnline(impl, now),
      priv: isHighPrivilege(impl.Username) ? 'high' : 'normal',
      color: colorsByAgent?.[impl.ID] || '',
      tags: tagsByAgent?.[impl.ID] || [],
      direction,
    },
    selected: selectedAgentIDs.includes(impl.ID),
  }
}

function pivotListenerFor(parentID, c2, pivotListeners) {
  const remote = c2.endpoint.toLowerCase()
  const parentListeners = (pivotListeners || []).filter((l) => l.ParentSessionID === parentID)
  return parentListeners.find((l) =>
    (l.Pivots || []).some((p) => String(p.RemoteAddress || '').toLowerCase() === remote)) ||
    (parentListeners.length === 1 ? parentListeners[0] : null)
}

function pivotDetails(parentID, c2, pivotListeners) {
  const listener = pivotListenerFor(parentID, c2, pivotListeners)
  if (!listener?.BindAddress) return { id: `p_${parentID}_${c2.key}`, label: c2.label }
  const port = endpointPort(listener.BindAddress)
  const endpoint = port ? `${endpointHost(c2.endpoint)}:${port}` : listener.BindAddress
  return { id: `p_${parentID}_${listener.ID}`, label: `${listener.Type || 'TCP'} ${endpoint}` }
}

// addC2Links pushes one listener node + one edge per unique C2 endpoint,
// deduped so multiple agents sharing a listener don't spawn duplicate boxes.
export function addC2Links(rawNodes, rawEdges, ctx) {
  const { allAgents, index, parentBySession, direction, pivotListeners } = ctx
  const { allAgentIds, agentsByAddress, agentsByName } = index
  const seenListeners = new Set()
  const seenPivotListeners = new Set()
  for (const impl of allAgents) {
    const kind = impl._kind
    const c2 = c2Details(impl.ActiveC2 || impl.RemoteAddress)
    const candidateParent = parentBySession.get(impl.ID) ||
      pivotParentFromC2(c2, agentsByAddress, agentsByName)
    const parentID = candidateParent !== impl.ID && allAgentIds.has(candidateParent)
      ? candidateParent
      : ''
    const sourceId = pushC2Node(rawNodes, rawEdges, {
      c2, parentID, direction, pivotListeners, seenListeners, seenPivotListeners,
    })
    rawEdges.push(c2Edge(sourceId, impl.ID, kind, parentID))
  }
}

function pushC2Node(rawNodes, rawEdges, opts) {
  const { c2, parentID, direction, pivotListeners, seenListeners, seenPivotListeners } = opts
  if (parentID) {
    const pivot = pivotDetails(parentID, c2, pivotListeners)
    if (!seenPivotListeners.has(pivot.id)) {
      seenPivotListeners.add(pivot.id)
      rawNodes.push({ id: pivot.id, w: LISTENER_W, h: LISTENER_H, data: { variant: 'listener', label: pivot.label, direction } })
      rawEdges.push({
        id: `e_${parentID}_${pivot.id}`, source: parentID, target: pivot.id,
        style: 'stroke:var(--color-success-500);stroke-width:3', animated: false,
      })
    }
    return pivot.id
  }
  const listenerID = `l_${c2.key}`
  if (!seenListeners.has(listenerID)) {
    seenListeners.add(listenerID)
    rawNodes.push({ id: listenerID, w: LISTENER_W, h: LISTENER_H, data: { variant: 'listener', label: c2.label, direction } })
    rawEdges.push({
      id: `e_ts_${listenerID}`, source: 'ts', target: listenerID,
      style: 'stroke:var(--color-line);stroke-dasharray:4', animated: false,
    })
  }
  return listenerID
}

function c2Edge(sourceId, targetID, kind, parentID) {
  const width = parentID ? 3 : 2
  const style = kind === 'beacon'
    ? `stroke:var(--color-beacon-500);stroke-width:${width};stroke-dasharray:5`
    : `stroke:var(--color-success-500);stroke-width:${width}`
  return { id: `e_${sourceId}_${targetID}`, source: sourceId, target: targetID, animated: false, style }
}

export function addDiscoveryNodes(rawNodes, rawEdges, ctx) {
  const { allAgentIds, discoveries, direction, selectedDiscoveryKeys, colorsByEntity } = ctx
  for (const device of discoveries || []) {
    const observerIDs = (device.observerIDs || [device.agentID]).filter((id) => allAgentIds.has(id))
    if (observerIDs.length === 0) continue
    const key = device.key || `${device.agentID}|${device.ip}`
    const deviceID = `d_${key}`
    const entityID = device.ip || device.agentID
    rawNodes.push({
      id: deviceID, w: DEVICE_W, h: DEVICE_H,
      data: {
        variant: 'device',
        entityID,
        ip: device.ip, mac: device.mac || '',
        hostname: device.hostname || '', vendor: device.vendor || '',
        osHint: device.osHint || '', ttl: device.ttl || 0,
        method: device.method || 'discovery', lastSeen: device.lastSeen || 0,
        agentID: observerIDs[0], observerIDs, key, direction,
        color: colorsByEntity?.[`device:${entityID}`] || '',
      },
      selected: selectedDiscoveryKeys.includes(key),
    })
    for (const observerID of observerIDs) {
      rawEdges.push({
        id: `e_${observerID}_${deviceID}`, source: observerID, target: deviceID,
        style: 'stroke:var(--color-device-500);stroke-width:1.5;stroke-dasharray:5',
        animated: false,
      })
    }
  }
}
