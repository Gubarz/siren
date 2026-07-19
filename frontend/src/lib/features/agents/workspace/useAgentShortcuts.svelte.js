// lib/features/agents/workspace/useAgentShortcuts.svelte.js
import { register as registerShortcut, unregister as unregisterShortcut } from '$stores/ui/shortcuts.svelte.js'
import { selection } from '$stores/ui/selection.svelte.js'
import { agentTabs } from '$stores/agentTabs.svelte.js'

export function useAgentShortcuts({ activeView, filteredData, filterInput }) {
  // We completely encapsulate the cursor state here!
  let cursor = $state(0)

  // 1. Keep the cursor safely bounded when the array changes
  $effect(() => {
    const data = filteredData()
    if (data.length === 0) {
      cursor = 0
      return
    }
    if (cursor > data.length - 1) {
      cursor = data.length - 1
    }
  })

  function syncCursorSelection() {
    const data = filteredData()
    const agent = data[cursor]
    if (!agent?.ID) return
    selection.select('agent', agent.ID, false)
  }

  // Best-effort scroll into view whenever the cursor moves. Post-DOM
  // $effect timing means the `bg-row-selected` class is already on the
  // right row when this runs, so no queueMicrotask/setTimeout dance.
  $effect(() => {
    cursor
    const el = document.querySelector('tbody tr.bg-row-selected')
    el?.scrollIntoView({ block: 'nearest' })
  })

  function moveCursor(delta) {
    const data = filteredData()
    if (data.length === 0) return
    cursor = Math.max(0, Math.min(data.length - 1, cursor + delta))
    syncCursorSelection()
  }

  function moveCursorTo(index) {
    const data = filteredData()
    if (data.length === 0) return
    cursor = Math.max(0, Math.min(data.length - 1, index))
    syncCursorSelection()
  }

  // 2. Automatically mount and unmount the shortcuts
  $effect(() => {
    const agentOnly = (fn) => () => { if (activeView() === 'agents') fn() }

    registerShortcut('J', agentOnly(() => moveCursor(+1)), 'Move cursor down', 'agents')
    registerShortcut('K', agentOnly(() => moveCursor(-1)), 'Move cursor up', 'agents')
    registerShortcut('G', agentOnly(() => moveCursorTo(0)), 'Jump to first', 'agents')
    registerShortcut('Shift+G', agentOnly(() => moveCursorTo(filteredData().length - 1)), 'Jump to last', 'agents')
    
    registerShortcut('/', agentOnly(() => filterInput()?.focus()), 'Focus filter box', 'agents')
    
    registerShortcut('Enter', agentOnly(() => {
      const data = filteredData()
      if (data.length === 0) return
      const id = data[cursor]?.ID
      if (id) agentTabs.openTab(id, 'console')
    }), 'Interact with agent', 'agents')
    
    registerShortcut('Escape', agentOnly(() => {
      const input = filterInput()
      if (document.activeElement === input) {
        input.blur()
      } else {
        selection.clear()
      }
    }), 'Blur filter or clear', 'agents')

    // Svelte 5 effect cleanups replace the onDestroy() boilerplate natively
    return () => {
      unregisterShortcut('J')
      unregisterShortcut('K')
      unregisterShortcut('G')
      unregisterShortcut('Shift+G')
      unregisterShortcut('/')
      unregisterShortcut('Enter')
      unregisterShortcut('Escape')
    }
  })
}