import { workspaceState } from '../workspaceState.svelte.js'

const validViews = new Set(['agents', 'automation', 'server', 'settings'])
const validServerTabs = new Set([
  'console',
  'listeners',
  'builds',
  'profiles',
  'loot',
  'creds',
  'operators',
  'cases',
  'websites',
  'http-c2',
  'traffic-encoders',
  'shellcode',
  'hosts',
  'canaries',
  'monitors',
  'cracking',
  'build-farm',
  'server',
  'events',
])

class Navigation {
  activeView = $state('agents')
  serverTab = $state('listeners')

  setView(activeView, options = {}) {
    if (!validViews.has(activeView)) return
    this.activeView = activeView
    workspaceState.set('activeView', activeView)
    if (options.serverTab != null) {
      this.serverTab = options.serverTab
      workspaceState.set('serverTab', options.serverTab)
    }
  }

  setServerTab(serverTab) {
    if (!validServerTabs.has(serverTab)) return
    if (this.serverTab !== serverTab) {
      this.serverTab = serverTab
      workspaceState.set('serverTab', serverTab)
    }
  }

  openServerTab(serverTab) {
    if (!validServerTabs.has(serverTab)) return
    if (this.activeView === 'server' && this.serverTab === serverTab) return
    this.activeView = 'server'
    this.serverTab = serverTab
    workspaceState.set('activeView', 'server')
    workspaceState.set('serverTab', serverTab)
  }

  hydrateFromWorkspace() {
    if (validViews.has(workspaceState.activeView)) {
      this.activeView = workspaceState.activeView
    }
    if (validServerTabs.has(workspaceState.serverTab)) {
      this.serverTab = workspaceState.serverTab
    }
  }
}

export const navigation = new Navigation()
