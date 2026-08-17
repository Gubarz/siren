import { CloseShell, StartShell } from '../api/console.js'
import { config } from './config.svelte.js'
import { dialog } from './ui/dialog.svelte.js'
import { errorMessage } from '../utils/errors.js'
import { shellPath } from '../utils/shell.js'

export const TAB_META = {
  console: { icon: 'terminal', label: 'Console' },
  tasks: { icon: 'list', label: 'Tasks' },
  fileBrowser: { icon: 'folder', label: 'Files' },
  processExplorer: { icon: 'cpu', label: 'Processes' },
  ifconfig: { icon: 'network-wired', label: 'Ifconfig' },
  netstat: { icon: 'list', label: 'Netstat' },
  registryBrowser: { icon: 'database', label: 'Registry' },
  screenshot: { icon: 'monitor', label: 'Screenshot' },
  grep: { icon: 'search', label: 'Grep' },
  services: { icon: 'cog', label: 'Services' },
  tunneling: { icon: 'network-wired', label: 'Tunnels' },
  extensions: { icon: 'package', label: 'Extensions' },
  memfiles: { icon: 'hard-drive', label: 'Memfiles' },
  privileges: { icon: 'key', label: 'Privileges' },
  env: { icon: 'braces', label: 'Env' },
  'wg-tunnels': { icon: 'shield', label: 'WG Tunnels' },
}

function tabLabel(sessionId, type) {
  const hash = sessionId?.slice(0, 8) || 'unknown'
  const meta = TAB_META[type]
  if (meta) return `${hash} - ${meta.label}`
  if (type.startsWith('shell-')) return `${hash} - Shell`
  return `${hash} - ${type}`
}

function normalizePane(pane = { tabs: [], activeTabId: '' }) {
  const tabs = pane.tabs || []
  const activeTabId = tabs.some((tab) => tab.id === pane.activeTabId)
    ? pane.activeTabId
    : (tabs[tabs.length - 1]?.id || '')
  return { tabs, activeTabId }
}

function normalizeState(state, preferredFocus = state.focusPane) {
  const left = normalizePane(state.panes.left)
  const right = normalizePane(state.panes.right)
  if (left.tabs.length === 0 && right.tabs.length > 0) {
    return { ...state, panes: { left: right }, focusPane: 'left' }
  }

  const panes = { left }
  if (right.tabs.length > 0) panes.right = right

  let focusPane = panes[preferredFocus] ? preferredFocus : 'left'

  return { ...state, panes, focusPane }
}

function isShellTab(tab) {
  return tab?.type?.startsWith('shell-')
}

function normalizeShell(shell) {
  if (!shell) return null
  const id = shell.id || shell.ID
  if (!id) return null
  return {
    ...shell,
    id,
    sessionID: shell.sessionID || shell.SessionID || shell.sessionId || '',
    path: shell.path || shell.Path || '',
    pid: shell.pid ?? shell.PID ?? 0,
    pty: shell.pty ?? shell.PTY ?? false,
  }
}

function panesHaveTabType(panes, type) {
  return Object.values(panes).some((pane) => pane.tabs.some((tab) => tab.type === type))
}

function decrementDetachedCount(counts, type) {
  const next = { ...counts }
  const count = next[type] || 0
  if (count <= 1) delete next[type]
  else next[type] = count - 1
  return next
}

function closeRemovedShells(shellsByID, removedTabs, panesAfterRemoval, detachedTypeCounts = {}) {
  let nextShells = shellsByID
  for (const tab of removedTabs) {
    if (!isShellTab(tab) || panesHaveTabType(panesAfterRemoval, tab.type) || detachedTypeCounts[tab.type] > 0) continue
    CloseShell(tab.type).catch(() => {})
    if (nextShells === shellsByID) nextShells = { ...shellsByID }
    delete nextShells[tab.type]
  }
  return nextShells
}

const INITIAL = {
  panes: { left: { tabs: [], activeTabId: '' } },
  focusPane: 'left',
  shellsByID: {},
  detachedTypeCounts: {},
}

const SHELL_LAUNCH_DEDUPE_MS = 2000

class AgentTabs {
  #state = $state({ ...INITIAL })
  #shellLaunches = new Map()
  #recentShellLaunches = new Map()
  #adultCheck = null

  get panes() { return this.#state.panes }
  get focusPane() { return this.#state.focusPane }
  get shellsByID() { return this.#state.shellsByID }
  get detachedTypeCounts() { return this.#state.detachedTypeCounts }
  get state() { return this.#state }

  #update(fn) {
    const next = fn(this.#state)
    if (next !== undefined) this.#state = next
  }

  openTab(sessionId, type, pane, meta = null) {
    if (!sessionId || !type) return
    const id = `${sessionId}-${type}`
    this.#update((s) => {
      const targetPane = pane || (s.panes[s.focusPane] ? s.focusPane : 'left')
      const paneState = s.panes[targetPane] || { tabs: [], activeTabId: '' }
      const existing = paneState.tabs.find((t) => t.id === id)
      if (existing) {
        return {
          ...s,
          panes: { ...s.panes, [targetPane]: { ...paneState, activeTabId: id } },
          focusPane: targetPane,
        }
      }
      const label = tabLabel(sessionId, type)
      const newTab = { id, sessionId, type, label, meta }
      return normalizeState({
        ...s,
        panes: {
          ...s.panes,
          [targetPane]: { tabs: [...paneState.tabs, newTab], activeTabId: id },
        },
        focusPane: targetPane,
      }, targetPane)
    })
  }

  openOrUpdateTab(sessionId, type, meta = null) {
    if (!sessionId || !type) return
    const id = `${sessionId}-${type}`
    this.#update((s) => {
      let foundPane = null
      let foundTab = null
      for (const paneId of Object.keys(s.panes)) {
        const pane = s.panes[paneId]
        if (!pane) continue
        const tab = pane.tabs.find((t) => t.id === id)
        if (tab) { foundPane = paneId; foundTab = tab; break }
      }
      if (foundPane && foundTab) {
        return {
          ...s,
          panes: {
            ...s.panes,
            [foundPane]: {
              tabs: s.panes[foundPane].tabs.map((t) =>
                t.id === id ? { ...t, meta } : t
              ),
              activeTabId: id,
            },
          },
          focusPane: foundPane,
        }
      }
      const targetPane = s.panes[s.focusPane] ? s.focusPane : 'left'
      const paneState = s.panes[targetPane] || { tabs: [], activeTabId: '' }
      const label = tabLabel(sessionId, type)
      const newTab = { id, sessionId, type, label, meta }
      return normalizeState({
        ...s,
        panes: {
          ...s.panes,
          [targetPane]: { tabs: [...paneState.tabs, newTab], activeTabId: id },
        },
        focusPane: targetPane,
      }, targetPane)
    })
  }

  selectTab(paneId, tabId) {
    this.#update((s) => {
      const pane = s.panes[paneId]
      if (!pane) return s
      if (!pane.tabs.some((tab) => tab.id === tabId)) return normalizeState(s)
      return {
        ...s,
        panes: { ...s.panes, [paneId]: { ...pane, activeTabId: tabId } },
        focusPane: paneId,
      }
    })
  }

  setFocusPane(paneId) {
    this.#update((s) => {
      if (!s.panes[paneId] || s.focusPane === paneId) return s
      return { ...s, focusPane: paneId }
    })
  }

  selectFocusedTabByIndex(index) {
    if (!Number.isInteger(index) || index < 0) return
    this.#update((s) => {
      const paneId = s.panes[s.focusPane] ? s.focusPane : 'left'
      const pane = s.panes[paneId]
      const tab = pane?.tabs?.[index]
      if (!tab) return s
      return {
        ...s,
        panes: { ...s.panes, [paneId]: { ...pane, activeTabId: tab.id } },
        focusPane: paneId,
      }
    })
  }

  moveFocusedTabToPane(targetPane) {
    this.#update((s) => {
      if (!targetPane || !['left', 'right'].includes(targetPane)) return s
      const fromPane = s.panes[s.focusPane] ? s.focusPane : 'left'
      if (fromPane === targetPane) return s
      const from = s.panes[fromPane]
      const tab = from?.tabs?.find((t) => t.id === from.activeTabId)
      if (!tab) return s

      const to = s.panes[targetPane] || { tabs: [], activeTabId: '' }
      let targetTabs = to.tabs
      if (!to.tabs.some((t) => t.id === tab.id)) {
        targetTabs = [...to.tabs, tab]
      }

      const sourceTabs = from.tabs.filter((t) => t.id !== tab.id)
      const newPanes = {
        ...s.panes,
        [fromPane]: {
          tabs: sourceTabs,
          activeTabId: sourceTabs[sourceTabs.length - 1]?.id || '',
        },
        [targetPane]: { tabs: targetTabs, activeTabId: tab.id },
      }
      if (fromPane === 'right' && sourceTabs.length === 0) {
        delete newPanes.right
      }

      return normalizeState({ ...s, panes: newPanes, focusPane: targetPane }, targetPane)
    })
  }

  closeTab(paneId, tabId) {
    this.#update((s) => {
      const pane = s.panes[paneId]
      if (!pane) return s
      const tabs = pane.tabs.filter((t) => t.id !== tabId)
      const closedTab = pane.tabs.find((t) => t.id === tabId)
      const wasActive = pane.activeTabId === tabId
      const newActive = wasActive ? (tabs[tabs.length - 1]?.id || '') : pane.activeTabId
      const newPanes = { ...s.panes, [paneId]: { tabs, activeTabId: newActive } }
      let focusPane = s.focusPane
      if (paneId === 'right' && tabs.length === 0) {
        delete newPanes.right
        focusPane = 'left'
      } else if (focusPane === paneId && tabs.length === 0 && newPanes.right?.tabs?.length) {
        focusPane = 'right'
      } else if (!newPanes[focusPane]) {
        focusPane = 'left'
      }
      const next = normalizeState({ ...s, panes: newPanes, focusPane }, focusPane)
      const shellsByID = closeRemovedShells(s.shellsByID, closedTab ? [closedTab] : [], next.panes, s.detachedTypeCounts)
      return { ...next, shellsByID }
    })
  }

  closeOthers(paneId, tabId) {
    this.#update((s) => {
      const pane = s.panes[paneId]
      if (!pane) return s
      const tab = pane.tabs.find((t) => t.id === tabId)
      if (!tab) return s
      const removed = pane.tabs.filter((t) => t.id !== tabId)
      const newPanes = { ...s.panes, [paneId]: { tabs: [tab], activeTabId: tabId } }
      const next = normalizeState({ ...s, panes: newPanes }, paneId)
      const shellsByID = closeRemovedShells(s.shellsByID, removed, next.panes, s.detachedTypeCounts)
      return { ...next, shellsByID }
    })
  }

  closeRight(paneId, tabId) {
    this.#update((s) => {
      const pane = s.panes[paneId]
      if (!pane) return s
      const idx = pane.tabs.findIndex((t) => t.id === tabId)
      if (idx === -1) return s
      const keep = pane.tabs.slice(0, idx + 1)
      const removed = pane.tabs.slice(idx + 1)
      const newActive = keep.some((t) => t.id === pane.activeTabId) ? pane.activeTabId : tabId
      const next = normalizeState({
        ...s,
        panes: { ...s.panes, [paneId]: { tabs: keep, activeTabId: newActive } },
      }, paneId)
      const shellsByID = closeRemovedShells(s.shellsByID, removed, next.panes, s.detachedTypeCounts)
      return { ...next, shellsByID }
    })
  }

  detachTab(paneId, tabId) {
    let detached = false
    this.#update((s) => {
      const pane = s.panes[paneId]
      const tab = pane?.tabs.find((candidate) => candidate.id === tabId)
      if (!tab) return s

      detached = true
      const tabs = pane.tabs.filter((candidate) => candidate.id !== tabId)
      const newPanes = {
        ...s.panes,
        [paneId]: {
          tabs,
          activeTabId: pane.activeTabId === tabId
            ? (tabs[tabs.length - 1]?.id || '')
            : pane.activeTabId,
        },
      }
      if (paneId === 'right' && tabs.length === 0) delete newPanes.right

      const detachedTypeCounts = {
        ...s.detachedTypeCounts,
        [tab.type]: (s.detachedTypeCounts[tab.type] || 0) + 1,
      }
      const focusPane = newPanes[s.focusPane] ? s.focusPane : 'left'
      return normalizeState({ ...s, panes: newPanes, focusPane, detachedTypeCounts }, focusPane)
    })
    return detached
  }

  restoreDetachedTab(envelope) {
    const tab = envelope?.tab
    if (!tab?.id || !tab?.type || !tab?.sessionId) return false
    if (envelope.shell) this.registerShell(envelope.shell)

    let restored = false
    this.#update((s) => {
      const detachedTypeCounts = decrementDetachedCount(s.detachedTypeCounts, tab.type)
      for (const [paneId, pane] of Object.entries(s.panes)) {
        if (!pane.tabs.some((candidate) => candidate.id === tab.id)) continue
        restored = true
        return {
          ...s,
          detachedTypeCounts,
          panes: { ...s.panes, [paneId]: { ...pane, activeTabId: tab.id } },
          focusPane: paneId,
        }
      }

      restored = true
      const paneId = s.panes[s.focusPane] ? s.focusPane : 'left'
      const pane = s.panes[paneId] || { tabs: [], activeTabId: '' }
      return normalizeState({
        ...s,
        detachedTypeCounts,
        panes: {
          ...s.panes,
          [paneId]: { tabs: [...pane.tabs, tab], activeTabId: tab.id },
        },
        focusPane: paneId,
      }, paneId)
    })
    return restored
  }

  releaseDetachedTab(type) {
    if (!type) return
    this.#update((s) => {
      const detachedTypeCounts = decrementDetachedCount(s.detachedTypeCounts, type)
      let shellsByID = s.shellsByID
      if (type.startsWith('shell-') && !panesHaveTabType(s.panes, type) && !detachedTypeCounts[type]) {
        CloseShell(type).catch(() => {})
        shellsByID = { ...s.shellsByID }
        delete shellsByID[type]
      }
      return { ...s, detachedTypeCounts, shellsByID }
    })
  }

  moveTab(tabId, fromPane, toPane, toIndex = null) {
    this.#update((s) => {
      const from = s.panes[fromPane]
      if (!from) return s
      const tab = from.tabs.find((t) => t.id === tabId)
      if (!tab) return s
      const newFromTabs = from.tabs.filter((t) => t.id !== tabId)
      const newFromActive = from.activeTabId === tabId
        ? (newFromTabs[newFromTabs.length - 1]?.id || '')
        : from.activeTabId
      const to = s.panes[toPane] || { tabs: [], activeTabId: '' }
      let newToTabs = to.tabs
      if (!to.tabs.some((t) => t.id === tabId)) {
        newToTabs = [...to.tabs]
        const insertAt = toIndex == null
          ? newToTabs.length
          : Math.max(0, Math.min(toIndex, newToTabs.length))
        newToTabs.splice(insertAt, 0, tab)
      }
      const newPanes = {
        ...s.panes,
        [fromPane]: { tabs: newFromTabs, activeTabId: newFromActive },
        [toPane]: { tabs: newToTabs, activeTabId: tabId },
      }
      if (fromPane === 'right' && newFromTabs.length === 0) {
        delete newPanes.right
      }
      return normalizeState({ ...s, panes: newPanes, focusPane: newPanes[toPane] ? toPane : 'left' }, toPane)
    })
  }

  reorderTab(paneId, tabId, toIndex) {
    this.#update((s) => {
      const pane = s.panes[paneId]
      if (!pane) return s
      const tabs = [...pane.tabs]
      const fromIdx = tabs.findIndex((t) => t.id === tabId)
      if (fromIdx === -1) return s
      const targetSlot = Math.max(0, Math.min(toIndex, pane.tabs.length))
      if (targetSlot === fromIdx || targetSlot === fromIdx + 1) return s
      const [tab] = tabs.splice(fromIdx, 1)
      const adjustedTo = targetSlot > fromIdx ? targetSlot - 1 : targetSlot
      tabs.splice(adjustedTo, 0, tab)
      return {
        ...s,
        panes: { ...s.panes, [paneId]: { tabs, activeTabId: pane.activeTabId } },
      }
    })
  }

  registerShell(shell) {
    const normalized = normalizeShell(shell)
    if (!normalized) return null
    this.#update((s) => ({
      ...s,
      shellsByID: { ...s.shellsByID, [normalized.id]: normalized },
    }))
    return normalized
  }

  #rememberShellLaunch(launchKey, shell) {
    const existing = this.#recentShellLaunches.get(launchKey)
    if (existing?.timer) clearTimeout(existing.timer)
    const timer = setTimeout(() => {
      this.#recentShellLaunches.delete(launchKey)
    }, SHELL_LAUNCH_DEDUPE_MS)
    timer?.unref?.()
    this.#recentShellLaunches.set(launchKey, { shell, timer })
  }

  async #confirmAdultForShell() {
    if (config.confirmShellAdult === false) return true
    if (!this.#adultCheck) {
      this.#adultCheck = dialog.confirm('This is not opsec safe. Are you an adult?', 'Start Shell')
        .finally(() => {
          this.#adultCheck = null
        })
    }
    return this.#adultCheck
  }

  async launchShell(sessionID, tail = '') {
    const parsedPath = shellPath(tail)
    const launchKey = `${sessionID}\0${parsedPath}`
    const existingLaunch = this.#shellLaunches.get(launchKey)
    if (existingLaunch) return existingLaunch
    const recentLaunch = this.#recentShellLaunches.get(launchKey)
    if (recentLaunch?.shell) {
      this.openTab(sessionID, recentLaunch.shell.id)
      return recentLaunch.shell
    }

    const launch = (async () => {
      try {
        if (!(await this.#confirmAdultForShell())) return null
        const shell = this.registerShell(await StartShell(sessionID, parsedPath, true, 24, 100))
        if (!shell) throw new Error('shell opened without an id')
        this.openTab(sessionID, shell.id)
        this.#rememberShellLaunch(launchKey, shell)
        return shell
      } catch (err) {
        await dialog.alert(errorMessage(err, 'Could not open shell: '), 'Shell')
        return null
      } finally {
        this.#shellLaunches.delete(launchKey)
      }
    })()

    this.#shellLaunches.set(launchKey, launch)
    return launch
  }
}

export const agentTabs = new AgentTabs()
