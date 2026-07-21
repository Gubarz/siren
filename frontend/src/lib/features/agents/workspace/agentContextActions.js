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
  const items = [
    { icon: 'terminal', label: bulkLabel('Console', compatibleAgents.length), on: () => openTabs(agentTabs, compatibleAgents, 'console') },
  ]
  if (isBeacon) {
    const taskTargets = beaconAgents.length > 0 ? beaconAgents : [agent]
    items.push({ icon: 'list', label: bulkLabel('Tasks', taskTargets.length), on: () => openTabs(agentTabs, taskTargets, 'tasks') })
    items.push({ icon: 'info', label: 'Beacon Detail…', on: () => openBeaconDetail(agent) })
    items.push({ icon: 'terminal', label: 'Open Interactive Session', on: () => promoteBeacon(agent) })
    items.push({ icon: 'x', label: 'Close Interactive Session', on: () => demoteSession(agent) })
    return items
  }
  items.push({ icon: 'terminal', label: bulkLabel('New Shell', sessionAgents.length), on: () => sessionAgents.forEach(newShell) })
  items.push({ icon: 'folder', label: bulkLabel('File Browser', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'fileBrowser') })
  items.push({ icon: 'cpu', label: bulkLabel('Process Explorer', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'processExplorer') })
  items.push({ icon: 'network-wired', label: bulkLabel('Ifconfig', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'ifconfig') })
  items.push({ icon: 'list', label: bulkLabel('Netstat', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'netstat') })
  if (isWindows || windowsSessions.length > 0) {
    const registryTargets = windowsSessions.length > 0 ? windowsSessions : [agent]
    items.push({ icon: 'database', label: bulkLabel('Registry', registryTargets.length), on: () => openTabs(agentTabs, registryTargets, 'registryBrowser') })
  }
  items.push({ icon: 'monitor', label: bulkLabel('Take Screenshot', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'screenshot') })
  items.push({ icon: 'network-wired', label: bulkLabel('Tunnels', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'tunneling') })
  items.push({ icon: 'search', label: bulkLabel('Grep', sessionAgents.length), on: () => openTabs(agentTabs, sessionAgents, 'grep') })
  if (isWindows || windowsSessions.length > 0) {
    const serviceTargets = windowsSessions.length > 0 ? windowsSessions : [agent]
    items.push({ icon: 'cog', label: bulkLabel('Services', serviceTargets.length), on: () => openTabs(agentTabs, serviceTargets, 'services') })
  }
  return items
}

function buildDiscoveryActions({ agent, runDiscovery, promptPingSweep, clearDiscoveries }) {
  return [
    { icon: 'network-wired', label: 'Discover Neighbors (ARP)', on: () => runDiscovery(agent, 'arp') },
    { icon: 'search-location', label: 'Ping Sweep…', on: () => promptPingSweep(agent) },
    { icon: 'eraser', label: 'Clear Discoveries (this agent only)', on: () => clearDiscoveries(agent.ID, agent.Name || agent.Hostname || agent.ID) },
  ]
}

// Closed row-color palette — must stay in sync with internal/tags
// RowColorNames (the backend rejects anything outside the set).
const ROW_COLORS = ['red', 'orange', 'yellow', 'green', 'blue', 'purple', 'pink', 'gray']

function buildRowColorItems({ agent, targetAgents, setAgentRowColor }) {
  const targets = targetAgents.length > 0 ? targetAgents : [agent]
  const items = ROW_COLORS.map((name) => ({
    label: name[0].toUpperCase() + name.slice(1),
    on: () => setAgentRowColor(targets, name),
  }))
  items.push({ label: 'Clear', on: () => setAgentRowColor(targets, '') })
  return items
}

function buildManagementActions({ agent, targetAgents, renameAgent, openReconfigure, openEditTags, openComments, addToCase, setAgentRowColor }) {
  return [
    { icon: 'pen', label: 'Rename Agent…', on: () => renameAgent(agent) },
    { icon: 'sliders', label: 'Reconfigure…', on: () => openReconfigure(agent) },
    { icon: 'tag', label: 'Edit Tags…', on: () => openEditTags(agent) },
    { icon: 'message-square', label: 'Comments / Notes…', on: () => openComments('agent', agent.ID, agent.Name || agent.Hostname || agent.ID) },
    { icon: 'palette', label: 'Color', children: buildRowColorItems({ agent, targetAgents, setAgentRowColor }) },
    { icon: 'folder', label: 'Add to case…', on: () => addToCase({
      collection: 'agent', itemID: agent.ID,
      label: agent.Name || agent.Hostname || agent.ID,
    }) },
  ]
}

function buildDangerActions({ agent, isBeacon, killAgent, removeBeaconRecord }) {
  const items = [
    { icon: 'skull', label: 'Kill Agent', danger: true, on: () => killAgent(agent) },
  ]
  if (isBeacon) {
    items.push({ icon: 'trash', label: 'Remove Beacon Record', danger: true, on: () => removeBeaconRecord(agent) })
  }
  return items
}

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

// Assemble the full section list. Section dividers are only emitted when
// the following section actually has content — keeps the menu tight for
// beacons and pre-session-catalog states.
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

  const coreActions = buildCoreActions({
    agent, isBeacon, isWindows,
    targetAgents,
    agentTabs,
    openBeaconDetail: contextMenuHandlers.openBeaconDetail,
    promoteBeacon: contextMenuHandlers.promoteBeacon,
    demoteSession: contextMenuHandlers.demoteSession,
    newShell: contextMenuHandlers.newShell,
  })
  const discoveryActions = buildDiscoveryActions({
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
  const managementActions = buildManagementActions({
    agent,
    targetAgents,
    renameAgent: contextMenuHandlers.renameAgent,
    openReconfigure: contextMenuHandlers.openReconfigure,
    openEditTags: contextMenuHandlers.openEditTags,
    openComments: contextMenuHandlers.openComments,
    addToCase: contextMenuHandlers.addToCase,
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
    { items: coreActions },
    { divider: true },
    { items: discoveryActions },
  ]
  if (automationActions.length > 0) {
    sections.push({ divider: true }, { items: automationActions })
  }
  if (commandCategories.length > 0) {
    sections.push({ divider: true }, { title: 'Commands', items: commandCategories })
  }
  sections.push({ divider: true }, { items: managementActions })
  sections.push({ divider: true }, { items: dangerActions })
  return sections
}

export function buildDiscoveryContextSections({ device, selectedCount, openComments, removeDiscoveries, clearDiscoveries }) {
  const multi = selectedCount > 1
  return [
    { items: [
      { icon: 'copy', label: 'Copy IP', on: () => navigator.clipboard?.writeText(device.ip || '') },
      { icon: 'message-square', label: 'Comments / Notes…', on: () => openComments('device', device.ip || device.agentID, device.hostname || device.ip) },
    ] },
    { divider: true },
    { items: [
      { icon: 'trash', label: multi ? `Remove Selected (${selectedCount})` : 'Remove Discovered Device', danger: true, on: () => removeDiscoveries(device) },
      { icon: 'eraser', label: 'Clear ALL Discoveries (every agent)', danger: true, on: () => clearDiscoveries() },
    ] },
  ]
}
