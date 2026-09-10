// Shared context-menu handler assembly for the workspace agent menus.
// AgentTopPane's right-click menu and AgentGlobalMenu's Actions dropdown both
// feed buildAgentContextSections, so the common handlers — including lost
// session removal and id copying — live here once. Per-menu differences are
// passed in through `overrides`.

import { RemoveKnownAgent } from '../../../api/agents.js'
import { errorMessage } from '../../../utils/errors.js'

function copyAgentIDs(agents) {
  const ids = (agents || []).map((agent) => agent?.ID).filter(Boolean).join('\n')
  if (ids) navigator.clipboard?.writeText(ids)
}

function createRemoveLostAgents({ dialog, knownAgents, selection }) {
  return async function removeLostAgents(agents) {
    const targets = (agents || []).filter((agent) => agent?._lost)
    if (targets.length === 0) return
    const label = targets.length > 1
      ? `${targets.length} lost sessions`
      : `"${targets[0].Name || targets[0].Hostname || targets[0].ID}"`
    if (!(await dialog.confirm(`Remove ${label} from known agents?`, 'Confirm Remove'))) return
    try {
      await Promise.all(targets.map((target) => RemoveKnownAgent(target.ID)))
      await knownAgents.load()
      selection.clear()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Remove failed: '), 'Remove Lost Session')
    }
  }
}

export function buildAgentContextMenuHandlers({
  actions,
  agentTabs,
  addToCase,
  beaconDetail,
  dialog,
  knownAgents,
  selection,
  overrides = {},
}) {
  const {
    runDiscovery, promptPingSweep, clearDiscoveries,
    killAgent, killAgents, newShell, renameAgent, removeBeaconRecord, removeBeaconRecords,
    promoteBeacon, demoteSession, runAutomationRule,
  } = actions

  return {
    openBeaconDetail: (agent) => beaconDetail.show(agent.ID),
    promoteBeacon,
    demoteSession,
    newShell,
    runDiscovery,
    promptPingSweep,
    clearDiscoveries,
    renameAgent,
    runAutomationRule,
    addToCase: (payload) => addToCase.open(payload),
    killAgent,
    killAgents,
    removeBeaconRecord,
    removeBeaconRecords,
    copyID: copyAgentIDs,
    onremovelost: createRemoveLostAgents({ dialog, knownAgents, selection }),
    findAttackPaths: (agents) => {
      for (const target of agents) agentTabs.openTab(target.ID, 'bloodhound')
    },
    ...overrides,
  }
}
