const GUI_VIEWS = [
  { id: 'view-agents', name: 'Table', label: 'Switch to Table', icon: 'list', section: 'Views', view: 'agents', config: { agentViewMode: 'table' } },
  { id: 'view-network-graph', name: 'Network Graph', label: 'Open Network Graph', icon: 'network', section: 'Views', view: 'agents', config: { agentViewMode: 'graph' } },
  { id: 'view-server', name: 'Server', label: 'Switch to Server', icon: 'server', section: 'Views', view: 'server' },
  { id: 'view-automation', name: 'Automation', label: 'Switch to Automation', icon: 'bolt', section: 'Views', view: 'automation' },
  { id: 'view-settings', name: 'Settings', label: 'Switch to Settings', icon: 'cog', section: 'Views', view: 'settings' },
]

const GUI_PANELS = [
  { id: 'panel-listeners', name: 'Listeners', label: 'Open Listeners', icon: 'headphones', section: 'Panels', overlay: 'listeners' },
  { id: 'panel-generate', name: 'Generate Implant', label: 'Generate Implant', icon: 'factory', section: 'Panels', overlay: 'generate' },
  { id: 'panel-loot', name: 'Loot', label: 'Open Loot', icon: 'download', section: 'Panels', overlay: 'loot' },
  { id: 'panel-credentials', name: 'Credentials', label: 'Open Credentials', icon: 'key', section: 'Panels', overlay: 'credentials' },
  { id: 'panel-armory', name: 'Armory', label: 'Open Armory', icon: 'shield', section: 'Panels', overlay: 'armory' },
  { id: 'panel-pivots', name: 'Pivots', label: 'Open Pivots', icon: 'network', section: 'Panels', overlay: 'pivots' },
  { id: 'panel-operators', name: 'Operators', label: 'Open Operators', icon: 'users', section: 'Panels', overlay: 'operators' },
  { id: 'panel-gallery', name: 'Screenshot Gallery', label: 'Open Screenshot Gallery', icon: 'images', section: 'Panels', overlay: 'gallery' },
  { id: 'panel-builds', name: 'Builds', label: 'Open Builds', icon: 'factory', section: 'Panels', view: 'server', serverTab: 'builds' },
  { id: 'panel-profiles', name: 'Profiles', label: 'Open Profiles', icon: 'sliders', section: 'Panels', view: 'server', serverTab: 'profiles' },
  { id: 'panel-cases', name: 'Cases', label: 'Open Cases', icon: 'folder', section: 'Panels', view: 'server', serverTab: 'cases' },
  { id: 'panel-events', name: 'Events', label: 'Open Events', icon: 'history', section: 'Panels', view: 'server', serverTab: 'events' },
]

export const GUI_ACTIONS = [...GUI_VIEWS, ...GUI_PANELS]

export function runGuiAction(action, { navigation, overlays, config } = {}) {
  if (!action) return

  for (const [key, value] of Object.entries(action.config || {})) {
    config?.set?.(key, value)
  }

  if (action.serverTab) {
    if (navigation?.openServerTab) {
      navigation.openServerTab(action.serverTab)
    } else {
      navigation?.setView?.('server', { serverTab: action.serverTab })
    }
  } else if (action.view) {
    navigation?.setView?.(action.view)
  }

  if (action.overlay) {
    overlays?.open?.(action.overlay)
  }
}

export function toPaletteAction(action, deps) {
  return {
    id: action.id,
    label: action.label,
    description: action.description,
    icon: action.icon,
    section: action.section,
    tags: action.tags,
    on: () => runGuiAction(action, deps),
  }
}

export const GuiActionGroups = [
  {
    category: 'GUI',
    commands: GUI_ACTIONS.map((action) => ({ name: action.name, guiAction: action })),
  },
]
