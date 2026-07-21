// Pure layout math for NetworkGraph. Dagre call + position preservation +
// two signature helpers. No component state.

import dagre from '@dagrejs/dagre'
import { Position } from '@xyflow/svelte'
import { pivotParentMap } from '../../utils/agents.js'
import { isAgentOnline } from '../../utils/agents.js'

// ---- C2 endpoint parsing (used by node builders as well) ----

export function c2Details(value) {
  const raw = String(value || 'unknown')
  const [scheme, remainder = raw] = raw.includes('://') ? raw.split('://', 2) : ['tcp', raw]
  const chain = remainder.split('->').map((part) => part.trim()).filter(Boolean)
  const endpoint = (chain[0] || 'unknown').split('/')[0]
  return {
    key: `${scheme}_${endpoint}`,
    label: `${scheme.toUpperCase()} ${endpoint}`,
    endpoint,
    isPivot: chain.length > 1,
    parentName: chain.length > 1 ? chain[chain.length - 1] : '',
  }
}

export function pivotParentFromC2(c2, agentsByAddress, agentsByName) {
  if (!c2.isPivot) return ''
  const addressMatch = agentsByAddress.get(c2.endpoint.toLowerCase())
  if (addressMatch) return addressMatch
  return agentsByName.get(c2.parentName.toLowerCase()) || ''
}

export function endpointHost(endpoint) {
  const value = String(endpoint || '')
  const ipv6 = value.match(/^\[(.+)\](?::\d+)?$/)
  if (ipv6) return `[${ipv6[1]}]`
  return value.replace(/:\d+$/, '')
}

export function endpointPort(endpoint) {
  const match = String(endpoint || '').match(/:(\d+)$/)
  return match ? match[1] : ''
}

// ---- Dagre layout ----

export function layoutGraph(rawNodes, rawEdges, direction) {
  const g = new dagre.graphlib.Graph()
  const horizontal = direction === 'LR'
  g.setGraph({
    rankdir: direction,
    nodesep: horizontal ? 24 : 45,
    ranksep: 60,
    marginx: 20,
    marginy: 20,
  })
  g.setDefaultEdgeLabel(() => ({}))
  rawNodes.forEach((n) => g.setNode(n.id, { width: n.w, height: n.h }))
  rawEdges.forEach((e) => g.setEdge(e.source, e.target))
  dagre.layout(g)
  return rawNodes.map((n) => renderNode(n, g.node(n.id), horizontal))
}

function renderNode(raw, position, horizontal) {
  return {
    id: raw.id,
    type: 'box',
    data: raw.data,
    position: { x: position.x - raw.w / 2, y: position.y - raw.h / 2 },
    width: raw.w,
    height: raw.h,
    initialWidth: raw.w,
    initialHeight: raw.h,
    targetPosition: horizontal ? Position.Left : Position.Top,
    sourcePosition: horizontal ? Position.Right : Position.Bottom,
    draggable: true,
    selected: raw.selected ?? false,
  }
}

// Nodes the user dragged since the last layout keep their positions; new
// nodes anchored to a moved parent shift with it so the pivot subtree
// doesn't tear apart on an incremental change.
export function preservePositions(previousNodes, nextNodes, nextEdges) {
  const positions = new Map(previousNodes.map((n) => [n.id, n.position]))
  const layoutPositions = new Map(nextNodes.map((n) => [n.id, n.position]))
  const parentByNode = new Map(nextEdges.map((edge) => [edge.target, edge.source]))
  return nextNodes.map((node) => {
    const previous = previousNodes.find((current) => current.id === node.id)
    const position = positions.get(node.id)
    if (position) return { ...node, position, selected: previous?.selected ?? node.selected ?? false }
    const parentID = parentByNode.get(node.id)
    const oldParent = positions.get(parentID)
    const layoutParent = layoutPositions.get(parentID)
    if (!oldParent || !layoutParent) return node
    return {
      ...node,
      position: {
        x: node.position.x + oldParent.x - layoutParent.x,
        y: node.position.y + oldParent.y - layoutParent.y,
      },
    }
  })
}

// ---- Change-detection signatures ----

// topologySignature captures every input that changes the graph *content*
// (agent count, C2 addresses, pivots, discoveries). If it hasn't changed,
// we don't need to rebuild — the ticker `now` update alone shouldn't
// re-run layout.
export function topologySignature({ sessions, beacons, pivotGraph, pivotListeners, discoveries, now, colors }) {
  const pivotRelations = [...pivotParentMap(pivotGraph).entries()]
    .map(([child, parent]) => `${parent}>${child}`)
    .sort()
  return [
    ...(sessions || []).map((s) => `${s.ID}:${s.ActiveC2}:${s.RemoteAddress}:${s.Name}:${isAgentOnline(s, now)}`),
    ...(beacons || []).map((b) => `${b.ID}:${b.ActiveC2}:${b.RemoteAddress}:${b.Name}:${isAgentOnline(b, now)}`),
    ...pivotRelations,
    ...(pivotListeners || []).map((l) => `${l.ParentSessionID}:${l.ID}:${l.Type}:${l.BindAddress}`),
    ...(discoveries || []).map((d) =>
      `${d.key}:${(d.observerIDs || [d.agentID]).join(',')}:${d.ip}:${d.mac}:${d.hostname}:${d.vendor}:${d.osHint}:${d.ttl}:${d.method}:${d.lastSeen}`),
    ...Object.entries(colors || {}).map(([id, color]) => `c_${id}:${color}`),
  ].sort().join('|')
}

// layoutSignature captures only what dagre needs to run — node ids + edge
// pairs. If unchanged we can preserve dragged positions from the previous
// tick.
export function layoutSignature(rawNodes, rawEdges) {
  return [
    ...rawNodes.map((n) => `n:${n.id}`),
    ...rawEdges.map((e) => `e:${e.source}>${e.target}`),
  ].sort().join('|')
}
