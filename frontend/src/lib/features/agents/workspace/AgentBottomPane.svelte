<script>
  import { agentTabs, TAB_META } from '$stores/agentTabs.svelte.js'
  import { contextMenu } from '$stores/ui/contextMenu.svelte.js'
  import { toast } from '$stores/ui/toast.svelte.js'
  import { dispatchCommand } from '$stores/console.svelte.js'
  import { DetachAgentTab } from '$api/detachedTabs.js'
  import { errorMessage } from '$utils/errors.js'
  import SplitPane from '$components/patterns/SplitPane.svelte'
  import Tabs from '$components/patterns/Tabs.svelte'
  import { resolveTab } from './tabRegistry.js'
  import {
    isPointerAtWindowBoundary,
    pointerScreenPoint,
  } from './tabTearOut.js'

  let state = $derived(agentTabs)
  let dragOverPane = $state('')
  let dropTargetIndex = $state(-1)
  let indicatorLeft = $state(0)
  let draggingTab = $state(false)
  let pointerOutsideWindow = $state(false)
  let rootEl
  let dragRootRect = null
  let pointerDrag = null
  let lastDragPoint = null
  const dragHitCache = new Map()
  const pointerDragThreshold = 5

  // Context bag passed into resolveTab — every callback + slice of the
  // tabs store that a per-tab component might need. Kept as a $derived so
  // the reactive graph tracks the shell map without extra ceremony.
  let tabContext = $derived({
    shellsByID: state.shellsByID,
    isActive: (tab) => {
      const pane = paneOwningTab(tab.id)
      return pane?.activeTabId === tab.id
    },
    openShell: (sid, raw = '') => agentTabs.launchShell(sid, raw),
    runConsoleCommand: (sid, cmd) => {
      agentTabs.openTab(sid, 'console')
      dispatchCommand(sid, cmd)
    },
  })

  function paneOwningTab(tabId) {
    for (const paneId of Object.keys(state.panes)) {
      if (state.panes[paneId]?.tabs?.some((t) => t.id === tabId)) return state.panes[paneId]
    }
    return null
  }

  function tabIcon(type) {
    if (type?.startsWith('shell-')) return 'terminal'
    return TAB_META[type]?.icon || 'terminal'
  }

  function handleTabContextMenu(e, tab, paneId) {
    e.preventDefault()
    const pane = state.panes[paneId]
    if (!pane) return
    const items = []

    if (paneId === 'left') {
      items.push({ icon: 'arrow-right', label: 'Move to Right Pane', on: () => agentTabs.moveTab(tab.id, 'left', 'right') })
    } else if (paneId === 'right') {
      items.push({ icon: 'arrow-left', label: 'Move to Left Pane', on: () => agentTabs.moveTab(tab.id, 'right', 'left') })
    }

    const otherPane = paneId === 'left' ? 'right' : 'left'
    items.push({ icon: 'copy', label: 'Duplicate Tab', on: () => agentTabs.openTab(tab.sessionId, tab.type, otherPane) })
    items.push({ icon: 'external-link', label: 'Open in New Window', on: () => tearOutTab(tab, paneId) })

    items.push({ icon: 'x', label: 'Close', on: () => agentTabs.closeTab(paneId, tab.id) })

    if (pane.tabs.length > 1) {
      items.push({ icon: 'x', label: 'Close Others', on: () => agentTabs.closeOthers(paneId, tab.id) })
      const tabIndex = pane.tabs.findIndex((t) => t.id === tab.id)
      if (tabIndex < pane.tabs.length - 1) {
        items.push({ icon: 'x', label: 'Close Right', on: () => agentTabs.closeRight(paneId, tab.id) })
      }
    }

    contextMenu.open({
      x: e.clientX,
      y: e.clientY,
      sections: [{ items }],
    })
  }

  function handleTabPointerDown(e, tab, paneId) {
    if (e.button !== 0) return
    pointerDrag = {
      tab,
      paneId,
      pointerId: e.pointerId,
      startX: e.clientX,
      startY: e.clientY,
    }
    lastDragPoint = pointerScreenPoint(e)
    e.currentTarget.setPointerCapture?.(e.pointerId)
  }

  function handleTabPointerMove(e) {
    const drag = pointerDrag
    if (!drag || drag.pointerId !== e.pointerId) return
    lastDragPoint = pointerScreenPoint(e, lastDragPoint)

    if (!draggingTab) {
      if (Math.hypot(e.clientX - drag.startX, e.clientY - drag.startY) < pointerDragThreshold) return
      draggingTab = true
      dragRootRect = rootEl?.getBoundingClientRect() || null
      dragHitCache.clear()
      agentTabs.selectTab(drag.paneId, drag.tab.id)
    }

    e.preventDefault()
    pointerOutsideWindow = isPointerAtWindowBoundary(e, lastDragPoint)
    if (pointerOutsideWindow) {
      dragOverPane = ''
      dropTargetIndex = -1
      return
    }
    updatePointerDropTarget(e.clientX, e.clientY)
  }

  function handleTabPointerUp(e) {
    const drag = pointerDrag
    if (!drag || drag.pointerId !== e.pointerId) return
    const wasDragging = draggingTab
    const point = pointerScreenPoint(e, lastDragPoint)
    if (!wasDragging) {
      resetDragState()
      return
    }

    e.preventDefault()
    if (pointerOutsideWindow || isPointerAtWindowBoundary(e, point)) {
      finishPointerTearOut(point)
      return
    }

    const targetPaneId = dragOverPane
    const targetIndex = dropTargetIndex
    const createRight = isRightCreatePoint(e.clientX, e.clientY)
    resetDragState()

    if (createRight && drag.paneId !== 'right') {
      agentTabs.moveTab(drag.tab.id, drag.paneId, 'right')
    } else if (targetPaneId) {
      if (drag.paneId === targetPaneId) {
        agentTabs.reorderTab(targetPaneId, drag.tab.id, targetIndex)
      } else {
        agentTabs.moveTab(drag.tab.id, drag.paneId, targetPaneId, targetIndex)
      }
    }
  }

  function handleTabPointerCancel(e) {
    const drag = pointerDrag
    if (!drag || drag.pointerId !== e.pointerId) return
    resetDragState()
  }

  function finishPointerTearOut(point) {
    const drag = pointerDrag
    if (!drag) return
    resetDragState()
    void tearOutTab(drag.tab, drag.paneId, point)
  }

  function resetDragState() {
    draggingTab = false
    pointerOutsideWindow = false
    dragOverPane = ''
    dropTargetIndex = -1
    dragRootRect = null
    pointerDrag = null
    lastDragPoint = null
    dragHitCache.clear()
  }

  async function tearOutTab(tab, paneId, point = null) {
    const sourceTab = state.panes[paneId]?.tabs?.find((candidate) => candidate.id === tab.id)
    if (!sourceTab) return
    const shell = sourceTab.type?.startsWith('shell-')
      ? state.shellsByID[sourceTab.type] || null
      : null
    try {
      await DetachAgentTab(
        JSON.stringify({ tab: sourceTab, shell }),
        Math.round(point?.x || 0),
        Math.round(point?.y || 0),
      )
      agentTabs.detachTab(paneId, sourceTab.id)
    } catch (err) {
      toast.push({ variant: 'error', message: errorMessage(err, 'Could not open tab window: ') })
    }
  }

  function isRightCreatePoint(clientX, clientY) {
    if (state.panes.right?.tabs?.length) return false
    const rect = dragRootRect || rootEl?.getBoundingClientRect()
    return !!rect && clientX >= rect.left + rect.width / 2 &&
      clientY >= rect.top && clientY <= rect.bottom
  }

  function paneMetrics(wrapper, paneId) {
    const cached = dragHitCache.get(paneId)
    if (cached?.wrapper === wrapper) return cached
    const wrapRect = wrapper.getBoundingClientRect()
    const buttons = wrapper.querySelectorAll('[data-tab-button="true"]')
    const tabs = [...buttons].map((button) => {
      const rect = button.getBoundingClientRect()
      return {
        left: rect.left - wrapRect.left,
        right: rect.right - wrapRect.left,
        mid: rect.left + rect.width / 2,
      }
    })
    const metrics = { wrapper, tabs }
    dragHitCache.set(paneId, metrics)
    return metrics
  }

  function paneHit(wrapper, paneId, clientX) {
    const metrics = paneMetrics(wrapper, paneId)
    if (metrics.tabs.length === 0) return { index: 0, left: 0 }
    for (let i = 0; i < metrics.tabs.length; i++) {
      if (clientX < metrics.tabs[i].mid) {
        return { index: i, left: metrics.tabs[i].left }
      }
    }
    const last = metrics.tabs[metrics.tabs.length - 1]
    return { index: metrics.tabs.length, left: last.right }
  }

  function updateDropIndicator(paneId, hit) {
    if (dragOverPane !== paneId) dragOverPane = paneId
    if (dropTargetIndex !== hit.index) dropTargetIndex = hit.index
    if (indicatorLeft !== hit.left) indicatorLeft = hit.left
  }

  function updatePointerDropTarget(clientX, clientY) {
    if (isRightCreatePoint(clientX, clientY)) {
      dragOverPane = ''
      dropTargetIndex = -1
      return
    }
    const element = document.elementFromPoint(clientX, clientY)
    const wrapper = element?.closest?.('[data-agent-tab-strip]')
    if (!wrapper || !rootEl?.contains(wrapper)) {
      dragOverPane = ''
      dropTargetIndex = -1
      return
    }
    const paneId = wrapper.dataset.agentTabStrip
    updateDropIndicator(paneId, paneHit(wrapper, paneId, clientX))
  }

  function closeTab(paneId, tabId) {
    agentTabs.closeTab(paneId, tabId)
  }

  function selectTab(paneId, tabId) {
    agentTabs.selectTab(paneId, tabId)
  }

  function focusPane(paneId) {
    agentTabs.setFocusPane(paneId)
  }
</script>

<div bind:this={rootEl} class="flex-1 overflow-hidden relative flex">
  {#if draggingTab && !pointerOutsideWindow && !state.panes.right?.tabs?.length}
    <div
      class="absolute right-0 top-0 bottom-0 w-1/2 z-20 border-l-2 border-dashed border-brand bg-brand/10 flex items-center justify-center text-brand text-xs font-semibold pointer-events-none"
      role="none"
    >
      Drop to open right pane
    </div>
  {/if}

  {#snippet paneContent(paneId, pane)}
    <div
      class="flex flex-col h-full w-full min-w-0"
      role="presentation"
      onpointerdown={() => focusPane(paneId)}
      onfocusin={() => focusPane(paneId)}
    >
      <div
        data-agent-tab-strip={paneId}
        class="{dragOverPane === paneId ? 'bg-brand/10' : ''} flex bg-chrome border-b border-line shrink-0 transition-colors relative"
        role="none"
      >
        <div class="flex-1 min-w-0">
          <Tabs
            tabs={(pane?.tabs || []).map((t) => ({
              id: t.id,
              label: t.label,
              icon: tabIcon(t.type),
              onclose: () => closeTab(paneId, t.id),
              oncontextmenu: (e) => handleTabContextMenu(e, t, paneId),
              onpointerdown: (e) => handleTabPointerDown(e, t, paneId),
              onpointermove: handleTabPointerMove,
              onpointerup: handleTabPointerUp,
              onpointercancel: handleTabPointerCancel,
            }))}
            active={pane?.activeTabId || ''}
            onchange={(id) => selectTab(paneId, id)}
            variant="underline"
            class="flex-1 min-w-0"
          />
        </div>
        {#if dragOverPane === paneId}
          <div
            class="absolute top-0 bottom-0 w-1.5 bg-brand pointer-events-none z-10"
            style="left: {indicatorLeft}px"
          ></div>
        {/if}
      </div>
      <div class="flex-1 overflow-auto relative flex flex-col">
        {#if pane && pane.tabs.length > 0}
          {#each pane.tabs as tab (tab.id)}
            {@const resolved = resolveTab(tab, tabContext)}
            <div class="flex flex-col h-full" class:hidden={pane.activeTabId !== tab.id}>
              {#if resolved}
                {@const Component = resolved.component}
                <Component {...resolved.props} />
              {/if}
            </div>
          {/each}
        {:else}
          <div class="flex-1 flex items-center justify-center text-fg-muted">Double-click an agent to open a console, or right-click for more tabs.</div>
        {/if}
      </div>
    </div>
  {/snippet}

  {#if state.panes.right?.tabs?.length > 0}
    <SplitPane orientation="horizontal" minSize={20} maxSize={80}>
      {#snippet left()}
        {@render paneContent('left', state.panes.left)}
      {/snippet}
      {#snippet right()}
        {@render paneContent('right', state.panes.right)}
      {/snippet}
    </SplitPane>
  {:else}
    {@render paneContent('left', state.panes.left)}
  {/if}
</div>
