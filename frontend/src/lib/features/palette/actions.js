import { overlays } from '$stores/ui/overlays.svelte.js'
import { navigation } from '$stores/ui/navigation.svelte.js'
import { config } from '$stores/config.svelte.js'
import { sessions } from '$stores/resources/sessions.svelte.js'
import { beacons } from '$stores/resources/beacons.svelte.js'
import { agentTabs } from '$stores/agentTabs.svelte.js'
import { GUI_ACTIONS, toPaletteAction } from './GuiActions.js'
import { getRegisteredActions } from './registry.js'

// Palette actions module-level: acquire both resources once for the
// lifetime of the app so getActions() sees fresh data without the caller
// having to wire up its own lifecycle.
sessions.acquire()
beacons.acquire()

const staticActions = GUI_ACTIONS.map((action) => toPaletteAction(action, { navigation, overlays, config }))

function openAgentTabActions() {
  const actions = []
  for (const [paneId, pane] of Object.entries(agentTabs.panes || {})) {
    for (const tab of pane?.tabs || []) {
      actions.push({
        id: `agent-tab-${paneId}-${tab.id}`,
        label: `Tab: ${tab.label}`,
        description: `${paneId} pane - ${tab.type || 'agent tab'} - ${tab.sessionId || ''}`,
        icon: tab.type?.startsWith('shell-') ? 'terminal' : 'panel-left',
        section: 'Open Agent Tabs',
        tags: [tab.id, tab.type, tab.sessionId, paneId].filter(Boolean),
        on: () => {
          navigation.setView('agents')
          agentTabs.selectTab(paneId, tab.id)
        },
      })
    }
  }
  return actions
}

export function getActions() {
  const dynamic = []

  for (const s of sessions.data || []) {
    dynamic.push({
      id: `session-${s.ID}`,
      label: `Console: ${s.Hostname || s.ID}`,
      description: `${s.Username} @ ${s.RemoteAddress || ''}`,
      icon: 'terminal',
      section: 'Sessions',
      on: () => { navigation.setView('agents'); agentTabs.openTab(s.ID, 'console') },
    })
  }

  for (const b of beacons.data || []) {
    dynamic.push({
      id: `beacon-${b.ID}`,
      label: `Tasks: ${b.Hostname || b.ID}`,
      description: `${b.Username} @ ${b.RemoteAddress || ''}`,
      icon: 'list',
      section: 'Beacons',
      on: () => { navigation.setView('agents'); agentTabs.openTab(b.ID, 'tasks') },
    })
  }

  return [...staticActions, ...openAgentTabActions(), ...dynamic, ...getRegisteredActions()]
}
