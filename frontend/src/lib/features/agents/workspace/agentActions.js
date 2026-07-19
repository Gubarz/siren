// Async action handlers for the AgentTopPane context menu. Extracted so the
// component stays focused on state + template; every action is a factory
// that closes over the callbacks it needs, keeping call sites tiny.

import { KillAgent, RemoveBeacon, RenameAgent } from '../../../api/agents.js'
import { ClearNetworkDiscoveries, DiscoverNetwork } from '../../../api/discovery.js'
import { OpenBeaconSession, CloseBeaconSession } from '../../../api/operatorControls.js'
import { errorMessage } from '../../../utils/errors.js'

export function createAgentActions({ dialog, discoveries, agentTabs, selectedAgentIDsIncluding }) {
  async function runDiscovery(agent, method, cidr = '') {
    const targetIDs = selectedAgentIDsIncluding(agent)
    const results = await Promise.allSettled(targetIDs.map((id) => DiscoverNetwork(id, method, cidr)))
    const failures = results.flatMap((r, i) => r.status === 'rejected' ? [{ id: targetIDs[i], error: r.reason }] : [])
    await discoveries.refresh()
    if (failures.length > 0) {
      await dialog.alert(failures.map((f) => `${f.id}: ${errorMessage(f.error)}`).join('\n'), 'Network Discovery')
    }
  }

  async function promptPingSweep(agent) {
    const cidr = await dialog.prompt('IPv4 CIDR to sweep (maximum /24):', 'Ping Sweep', '192.168.1.0/24')
    if (!cidr) return
    await runDiscovery(agent, 'sweep', cidr)
  }

  async function clearDiscoveries(agentID = '', agentLabel = '') {
    const scope = agentID
      ? `Clear discoveries observed by "${agentLabel || agentID}"?`
      : 'Clear ALL discovered devices across every agent?'
    if (!(await dialog.confirm(scope, 'Confirm Clear'))) return
    try {
      await ClearNetworkDiscoveries(agentID)
      await discoveries.refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Clear failed: '), 'Network Discovery')
    }
  }

  async function killAgent(agent) {
    if (!(await dialog.confirm(`Kill agent "${agent.Name || agent.ID}"?`, 'Confirm Kill'))) return
    try {
      await KillAgent(agent.ID)
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Kill failed: '), 'Kill Agent')
    }
  }

  async function newShell(agent) {
    await agentTabs.launchShell(agent.ID, '')
  }

  async function renameAgent(agent) {
    const name = await dialog.prompt('New name:', 'Rename Agent', agent.Name || '')
    if (!name || name === agent.Name) return
    try {
      await RenameAgent(agent.ID, name)
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Rename failed: '), 'Rename Agent')
    }
  }

  async function removeBeaconRecord(agent) {
    if (!(await dialog.confirm(`Remove beacon record for "${agent.Name || agent.ID}"?`, 'Confirm Remove'))) return
    try {
      await RemoveBeacon(agent.ID)
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Remove failed: '), 'Remove Beacon')
    }
  }

  // promoteBeacon fires OpenBeaconSession; the resulting session arrives
  // asynchronously through the event stream — the caller doesn't need to
  // poll for it.
  async function promoteBeacon(agent) {
    try {
      await OpenBeaconSession({ beaconId: agent.ID, c2Urls: [], delay: 0 })
      await dialog.alert(
        `Interactive-session request queued for beacon ${agent.Name || agent.ID}. It'll appear as a session on the next callback.`,
        'Open Session',
      )
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Open session failed: '), 'Open Session')
    }
  }

  async function demoteSession(agent) {
    if (!(await dialog.confirm(
      `Close the interactive session for beacon ${agent.Name || agent.ID}? The beacon itself keeps polling.`,
      'Close Session',
    ))) return
    try {
      await CloseBeaconSession(agent.ID, '')
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Close session failed: '), 'Close Session')
    }
  }

  return {
    runDiscovery, promptPingSweep, clearDiscoveries,
    killAgent, newShell, renameAgent, removeBeaconRecord,
    promoteBeacon, demoteSession,
  }
}
