// Context-menu action builders for the Sessions/Beacons table.
// Kept out of AgentTopPane.svelte so component stays under the 300-line cap
// and tag/bulk-op actions have a natural place to plug in.
//
// Every builder is a pure function of the caller's callbacks — no store
// reads, no side effects at import time.

import { matchesAutomationTarget } from '../../../utils/automation.js'

function isWindowsAgent(agent) {
  return (agent.OS || '').toLowerCase() === 'windows'
}

function openTabs(agentTabs, agents, type) {
  for (const target of agents) {
    agentTabs.openTab(target.ID, type)
  }
}

function bulkLabel(label, count) {
  return count > 1 ? `${label} (${count})` : label
}

// ---------------------------------------------------------------------------
// Color palette
// ---------------------------------------------------------------------------

const ROW_COLORS = ['red', 'orange', 'yellow', 'green', 'blue', 'purple', 'pink', 'gray']

const COLOR_HEX_MAP = {
  red:    '#ef4444',
  orange: '#f97316',
  yellow: '#eab308',
  green:  '#22c55e',
  blue:   '#3b82f6',
  purple: '#a855f7',
  pink:   '#ec4899',
  gray:   '#9ca3af',
}

function buildColorPalette({ agent, targetAgents, setAgentRowColor }) {
  const targets = targetAgents.length > 0 ? targetAgents : [agent]
  const items = ROW_COLORS.map((name) => ({
    label: name[0].toUpperCase() + name.slice(1),
    color: COLOR_HEX_MAP[name],
    on: () => setAgentRowColor(targets, name),
  }))
  return items
}

function buildColorClearItem({ agent, targetAgents, setAgentRowColor }) {
  const targets = targetAgents.length > 0 ? targetAgents : [agent]
  return { label: 'Clear', color: '', on: () => setAgentRowColor(targets, '') }
}

// ---------------------------------------------------------------------------
// Core actions — split into top-level flat items and a "More" submenu.
// ---------------------------------------------------------------------------

function buildCoreActions({
  agent,
  isBeacon,
  isWindows,
  targetAgents,
  agentTabs,
  openBeaconDetail,
  promoteBeacon,
  demoteSession,
  newShell,
}) {
  const compatibleAgents = targetAgents.length > 0 ? targetAgents : [agent]
  const sessionAgents = compatibleAgents.filter((target) => target._kind !== 'beacon')
  const beaconAgents = compatibleAgents.filter((target) => target._kind === 'beacon')
  const windowsSessions = sessionAgents.filter(isWindowsAgent)

  if (isBeacon) {
    const taskTargets = beaconAgents.length > 0 ? beaconAgents : [agent]
    return {
      topLevel: [
        { icon: 'terminal', label: bulkLabel('Console', compatibleAgents.length), on: () => openTabs(agentTabs, compatibleAgents, 'console') },
        { icon: 'list', label: bulkLabel('Tasks', taskTargets.length), on: () => openTabs(agentTabs, taskTargets, 'tasks') },
        { icon: 'info', label: 'Beacon Detail…', on: () => openBeaconDetail(agent) },
        { icon: 'terminal', label: 'Open Interactive Session', on: () => promoteBeacon(agent) },
        { icon: 'x', label: 'Close Interactive Session', on: () => demoteSession(agent) },
      ],
      moreItems: [],
    }
  }

  const topLevel = [
    { icon: 'terminal', label: bulkLabel('Console', compatibleAgents.length), on: () => openTabs(agentTabs, compatibleAgents, 'console') },
    { icon: 'terminal', label: bulkLabel('New Shell', sessionAgents.length), on: () => sessionAgents.forEach(newShell) },
    { icon: 'folder', label: bulkLabel('File Browser', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'fileBrowser') },
    { icon: 'network-wired', label: bulkLabel('Tunnels', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'tunneling') },
    { icon: 'cpu', label: bulkLabel('Process Explorer', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'processExplorer') },
  ]

  if (isWindows || windowsSessions.length > 0) {
    const serviceTargets = windowsSessions.length > 0 ? windowsSessions : [agent]
    topLevel.push({ icon: 'cog', label: bulkLabel('Services', serviceTargets.length), on: () => openTabs(agentTabs, serviceTargets, 'services') })
  }

  const moreItems = [
    { icon: 'monitor', label: bulkLabel('Take Screenshot', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'screenshot') },
    { icon: 'search', label: bulkLabel('Grep', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'grep') },
  ]
  if (isWindows || windowsSessions.length > 0) {
    const registryTargets = windowsSessions.length > 0 ? windowsSessions : [agent]
    moreItems.push({ icon: 'database', label: bulkLabel('Registry', registryTargets.length), on: () => openTabs(agentTabs, registryTargets, 'registryBrowser') })
  }
  moreItems.push(
    { icon: 'list', label: bulkLabel('Netstat', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'netstat') },
    { icon: 'network-wired', label: bulkLabel('Ifconfig', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'ifconfig') },
  )
  if (isWindows || windowsSessions.length > 0) {
    moreItems.push({ icon: 'key', label: bulkLabel('Privileges', windowsSessions.length || 1), on: () => openTabs(agentTabs, windowsSessions.length > 0 ? windowsSessions : [agent], 'privileges') })
  }

  return { topLevel, moreItems }
}

// ---------------------------------------------------------------------------
// Discovery — now nested inside a "Discovery" submenu.
// ---------------------------------------------------------------------------

function buildDiscoveryActions({ agent, runDiscovery, promptPingSweep, clearDiscoveries }) {
  return [
    { icon: 'network-wired', label: 'Discover Neighbors (ARP)', on: () => runDiscovery(agent, 'arp') },
    { icon: 'search-location', label: 'Ping Sweep…', on: () => promptPingSweep(agent) },
    { icon: 'eraser', label: 'Clear Discoveries (this agent only)', on: () => clearDiscoveries(agent.ID, agent.Name || agent.Hostname || agent.ID) },
  ]
}

// ---------------------------------------------------------------------------
// Management — now nested inside a "Manage" submenu (color palette moved out).
// ---------------------------------------------------------------------------

function buildManagementActions({ agent, renameAgent, openReconfigure, openTags, openComments, addToCase }) {
  return [
    { icon: 'pen', label: 'Rename Agent…', on: () => renameAgent(agent) },
    { icon: 'sliders', label: 'Reconfigure…', on: () => openReconfigure(agent) },
    { icon: 'tag', label: 'Tags / Color…', on: () => openTags('agent', agent.ID, agent.Name || agent.Hostname || agent.ID) },
    { icon: 'message-square', label: 'Comments / Notes…', on: () => openComments('agent', agent.ID, agent.Name || agent.Hostname || agent.ID) },
    { icon: 'folder', label: 'Add to case…', on: () => addToCase({
      collection: 'agent', itemID: agent.ID,
      label: agent.Name || agent.Hostname || agent.ID,
    }) },
  ]
}

// ---------------------------------------------------------------------------
// Danger
// ---------------------------------------------------------------------------

function buildDangerActions({ agent, isBeacon, killAgent, removeBeaconRecord }) {
  const items = [
    { icon: 'skull', label: 'Kill Agent', danger: true, on: () => killAgent(agent) },
  ]
  if (isBeacon) {
    items.push({ icon: 'trash', label: 'Remove Beacon Record', danger: true, on: () => removeBeaconRecord(agent) })
  }
  return items
}

// ---------------------------------------------------------------------------
// Automation
// ---------------------------------------------------------------------------

function buildAutomationActions({ agent, automationRules, runAutomationRule, targetAgents }) {
  if (!automationRules || automationRules.length === 0) return []
  const targets = targetAgents.length > 0 ? targetAgents : [agent]

  const matchingRules = automationRules.filter(
    (rule) => rule.enabled !== false && matchesAutomationTarget(agent, rule)
  )

  if (matchingRules.length === 0) return []

  return [
    {
      icon: 'bolt',
      label: bulkLabel('Run Automation', targets.length),
      children: matchingRules.map((rule) => ({
        label: rule.name || 'Unnamed Rule',
        description: rule.trigger ? `Trigger: ${rule.trigger}` : '',
        on: () => runAutomationRule(rule, targets),
      })),
    },
  ]
}

// Turn the catalog categories into a nested-context-menu shape. Empty
// categories are dropped so a right-click never opens an empty submenu.
function buildCommandCategories({ catalog, targetIDs, executeAgentCommand }) {
  return catalog
    .map((cat) => ({
      icon: 'command',
      label: cat.category,
      children: cat.commands.map((cmd) => ({
        label: cmd.name,
        description: cmd.unavailable || cmd.description || '',
        on: () => executeAgentCommand(cmd, targetIDs),
      })),
    }))
    .filter((cat) => cat.children.length > 0)
}

// ---------------------------------------------------------------------------
// Assemble the full section list.
//   1. Top-level core actions (Console, New Shell, File Browser, Tunnels,
//      Process Explorer, Services)
//   2. "More" submenu (Screenshot, Grep, Registry, Netstat, Ifconfig, …)
//   3. "Discovery" submenu
//   4. Automation (conditional)
//   5. "Commands" category section
//   6. "Manage" submenu
//   7. Color palette (inline swatches)
//   8. Danger actions
// ---------------------------------------------------------------------------

export function buildAgentContextSections(ctx) {
  const {
    agent,
    isBeacon,
    isWindows,
    catalog,
    targetIDs,
    targetAgents,
    agentTabs,
    automationRules,
    contextMenuHandlers,
  } = ctx

  const { topLevel, moreItems } = buildCoreActions({
    agent, isBeacon, isWindows,
    targetAgents,
    agentTabs,
    openBeaconDetail: contextMenuHandlers.openBeaconDetail,
    promoteBeacon: contextMenuHandlers.promoteBeacon,
    demoteSession: contextMenuHandlers.demoteSession,
    newShell: contextMenuHandlers.newShell,
  })

  const discoveryItems = buildDiscoveryActions({
    agent,
    runDiscovery: contextMenuHandlers.runDiscovery,
    promptPingSweep: contextMenuHandlers.promptPingSweep,
    clearDiscoveries: contextMenuHandlers.clearDiscoveries,
  })

  const automationActions = buildAutomationActions({
    agent,
    automationRules,
    runAutomationRule: contextMenuHandlers.runAutomationRule,
    targetAgents,
  })

  const managementItems = buildManagementActions({
    agent,
    renameAgent: contextMenuHandlers.renameAgent,
    openReconfigure: contextMenuHandlers.openReconfigure,
    openTags: contextMenuHandlers.openTags,
    openComments: contextMenuHandlers.openComments,
    addToCase: contextMenuHandlers.addToCase,
  })

  const paletteItems = buildColorPalette({
    agent, targetAgents,
    setAgentRowColor: contextMenuHandlers.setAgentRowColor,
  })

  const paletteClear = buildColorClearItem({
    agent, targetAgents,
    setAgentRowColor: contextMenuHandlers.setAgentRowColor,
  })

  const dangerActions = buildDangerActions({
    agent, isBeacon,
    killAgent: contextMenuHandlers.killAgent,
    removeBeaconRecord: contextMenuHandlers.removeBeaconRecord,
  })

  const commandCategories = buildCommandCategories({
    catalog, targetIDs,
    executeAgentCommand: contextMenuHandlers.executeAgentCommand,
  })

  const sections = [
    { items: topLevel },
  ]

  if (moreItems.length > 0) {
    sections.push({
      items: [{ icon: 'ellipsis-vertical', label: 'More', children: moreItems }],
    })
    sections.push({ divider: true })
  }

  sections.push({
    items: [{ icon: 'network-wired', label: 'Discovery', children: discoveryItems }],
  })

  if (automationActions.length > 0) {
    sections.push({ divider: true }, { items: automationActions })
  }

  if (commandCategories.length > 0) {
    sections.push({ divider: true }, { title: 'Commands', items: commandCategories })
  }

  sections.push(
    { divider: true },
    {
      items: [{ icon: 'sliders', label: 'Manage', children: managementItems }],
    },
    { palette: true, items: paletteItems, clearItem: paletteClear },
    { divider: true },
    { items: dangerActions },
  )

  return sections
}

export function buildDiscoveryContextSections({ device, selectedCount, openComments, openTags, removeDiscoveries, clearDiscoveries }) {
  const multi = selectedCount > 1
  return [
    { items: [
      { icon: 'copy', label: 'Copy IP', on: () => navigator.clipboard?.writeText(device.ip || '') },
      { icon: 'tag', label: 'Tags / Color…', on: () => openTags('device', device.ip || device.agentID, device.hostname || device.ip) },
      { icon: 'message-square', label: 'Comments / Notes…', on: () => openComments('device', device.ip || device.agentID, device.hostname || device.ip) },
    ] },
    { divider: true },
    { items: [
      { icon: 'trash', label: multi ? `Remove Selected (${selectedCount})` : 'Remove Discovered Device', danger: true, on: () => removeDiscoveries(device) },
      { icon: 'eraser', label: 'Clear ALL Discoveries (every agent)', danger: true, on: () => clearDiscoveries() },
    ] },
  ]
}
