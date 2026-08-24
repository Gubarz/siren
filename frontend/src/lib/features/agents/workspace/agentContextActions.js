// Context-menu action builders for the Sessions/Beacons table.
// Kept out of AgentTopPane.svelte so component stays under the 300-line cap
// and tag/bulk-op actions have a natural place to plug in.
//
// Every builder is a pure function of the caller's callbacks — no store
// reads, no side effects at import time.

import { matchesAutomationTarget } from '../../../utils/automation.js'
import { ROW_COLORS, colorHex } from '../../../utils/agentColors.js'
import { TAB_META } from '../../../stores/agentTabs.svelte.js'

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

function tabAction(agentTabs, agents, type, count) {
  const meta = TAB_META[type]
  return {
    icon: meta?.icon ?? 'info',
    label: bulkLabel(meta?.label ?? type, count),
    on: () => openTabs(agentTabs, agents, type),
  }
}

// ---------------------------------------------------------------------------
// Color palette
// ---------------------------------------------------------------------------

function buildColorPalette({ agent, targetAgents, setAgentRowColor }) {
  const targets = targetAgents.length > 0 ? targetAgents : [agent]
  const items = ROW_COLORS.map((name) => ({
    label: name[0].toUpperCase() + name.slice(1),
    color: colorHex(name),
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
  hasInteractiveSession,
  targetAgents,
  agentTabs,
  openBeaconDetail,
  promoteBeacon,
  demoteSession,
  newShell,
  findAttackPaths,
}) {
  const compatibleAgents = targetAgents.length > 0 ? targetAgents : [agent]
  const sessionAgents = compatibleAgents.filter((target) => target._kind !== 'beacon')
  const beaconAgents = compatibleAgents.filter((target) => target._kind === 'beacon')
  const windowsSessions = sessionAgents.filter(isWindowsAgent)

  if (isBeacon) {
    const taskTargets = beaconAgents.length > 0 ? beaconAgents : [agent]
    const topLevel = [
      tabAction(agentTabs, compatibleAgents, 'console', compatibleAgents.length),
      tabAction(agentTabs, taskTargets, 'tasks', taskTargets.length),
      tabAction(agentTabs, taskTargets, 'processExplorer', taskTargets.length),
      { icon: 'info', label: 'Beacon Detail…', on: () => openBeaconDetail(agent) },
      { icon: 'terminal', label: 'Open Interactive Session', on: () => promoteBeacon(agent) },
      { icon: 'x', label: 'Close Interactive Session', disabled: !hasInteractiveSession, on: () => demoteSession(agent) },
      {
        icon: 'workflow',
        label: bulkLabel('Find attack paths', compatibleAgents.length),
        disabled: !findAttackPaths,
        on: () => findAttackPaths(compatibleAgents),
      },
    ]

    const moreItems = [
      tabAction(agentTabs, beaconAgents.length > 0 ? beaconAgents : [agent], 'screenshot', beaconAgents.length || 1),
      tabAction(agentTabs, beaconAgents.length > 0 ? beaconAgents : [agent], 'grep', beaconAgents.length || 1),
      tabAction(agentTabs, beaconAgents.length > 0 ? beaconAgents : [agent], 'env', beaconAgents.length || 1),
    ]
    if (isWindows) {
      moreItems.push(tabAction(agentTabs, [agent], 'registryBrowser', 1))
      moreItems.push(tabAction(agentTabs, [agent], 'services', 1))
    }
    moreItems.push(
      tabAction(agentTabs, beaconAgents.length > 0 ? beaconAgents : [agent], 'netstat', beaconAgents.length || 1),
      tabAction(agentTabs, beaconAgents.length > 0 ? beaconAgents : [agent], 'ifconfig', beaconAgents.length || 1),
    )
    if (isWindows) {
      moreItems.push(tabAction(agentTabs, [agent], 'privileges', 1))
    }

    return { topLevel, moreItems }
  }

  const topLevel = [
    tabAction(agentTabs, compatibleAgents, 'console', compatibleAgents.length),
    { icon: 'terminal', label: bulkLabel('New Shell', sessionAgents.length), on: () => sessionAgents.forEach(newShell) },
    tabAction(agentTabs, sessionAgents, 'fileBrowser', sessionAgents.length),
    tabAction(agentTabs, sessionAgents, 'tunneling', sessionAgents.length),
    tabAction(agentTabs, sessionAgents, 'processExplorer', sessionAgents.length),
    {
      icon: 'workflow',
      label: bulkLabel('Find attack paths', compatibleAgents.length),
      disabled: !findAttackPaths,
      on: () => findAttackPaths(compatibleAgents),
    },
  ]

  if (isWindows || windowsSessions.length > 0) {
    const serviceTargets = windowsSessions.length > 0 ? windowsSessions : [agent]
    topLevel.push(tabAction(agentTabs, serviceTargets, 'services', serviceTargets.length))
  }

  const moreItems = [
    tabAction(agentTabs, sessionAgents, 'screenshot', sessionAgents.length),
    tabAction(agentTabs, sessionAgents, 'grep', sessionAgents.length),
    tabAction(agentTabs, sessionAgents, 'env', sessionAgents.length),
  ]
  if (isWindows || windowsSessions.length > 0) {
    const registryTargets = windowsSessions.length > 0 ? windowsSessions : [agent]
    moreItems.push(tabAction(agentTabs, registryTargets, 'registryBrowser', registryTargets.length))
  }
  moreItems.push(
    tabAction(agentTabs, sessionAgents, 'netstat', sessionAgents.length),
    tabAction(agentTabs, sessionAgents, 'ifconfig', sessionAgents.length),
  )
  if (isWindows || windowsSessions.length > 0) {
    moreItems.push(tabAction(agentTabs, windowsSessions.length > 0 ? windowsSessions : [agent], 'privileges', windowsSessions.length || 1))
  }

  return { topLevel, moreItems }
}

// ---------------------------------------------------------------------------
// Discovery — now nested inside a "Discovery" submenu.
// ---------------------------------------------------------------------------

function buildDiscoveryActions({ agent, runDiscovery, promptPingSweep, clearDiscoveries, collectBloodHound }) {
  const items = [
    { icon: 'network-wired', label: 'Discover Neighbors (ARP)', on: () => runDiscovery(agent, 'arp') },
    { icon: 'search-location', label: 'Ping Sweep…', on: () => promptPingSweep(agent) },
  ]
  if (collectBloodHound) {
    items.push({ icon: 'workflow', label: 'Collect BloodHound data…', on: () => collectBloodHound(agent) })
  }
  items.push({ icon: 'eraser', label: 'Clear Discoveries (this agent only)', on: () => clearDiscoveries(agent.ID, agent.Name || agent.Hostname || agent.ID) })
  return items
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
export function buildCommandCategories({ catalog, targetIDs, executeAgentCommand, isDisabled }) {
  return catalog
    .map((cat) => ({
      icon: 'command',
      label: cat.category,
      children: cat.commands.map((cmd) => ({
        label: cmd.name,
        description: cmd.unavailable || cmd.description || '',
        disabled: isDisabled ? isDisabled(cmd) : false,
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
    hasInteractiveSession,
    catalog,
    targetIDs,
    targetAgents,
    agentTabs,
    automationRules,
    contextMenuHandlers,
  } = ctx

  const { topLevel, moreItems } = buildCoreActions({
    agent, isBeacon, isWindows,
    hasInteractiveSession: hasInteractiveSession ?? false,
    targetAgents,
    agentTabs,
    openBeaconDetail: contextMenuHandlers.openBeaconDetail,
    promoteBeacon: contextMenuHandlers.promoteBeacon,
    demoteSession: contextMenuHandlers.demoteSession,
    newShell: contextMenuHandlers.newShell,
    findAttackPaths: contextMenuHandlers.findAttackPaths,
  })

  const discoveryItems = buildDiscoveryActions({
    agent,
    runDiscovery: contextMenuHandlers.runDiscovery,
    promptPingSweep: contextMenuHandlers.promptPingSweep,
    clearDiscoveries: contextMenuHandlers.clearDiscoveries,
    collectBloodHound: contextMenuHandlers.collectBloodHound,
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
