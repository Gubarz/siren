<script>
  import { onMount } from 'svelte'
  import { register, unregister, handleKeydown, getActiveShortcuts } from '$stores/ui/shortcuts.svelte.js'
  import CommandPalette from '$features/palette/CommandPalette.svelte'
  import DocsModal from './DocsModal.svelte'
  import Modal from '$components/patterns/Modal.svelte'
  import Kbd from '$components/ui/Kbd.svelte'
  import { navigation } from '$stores/ui/navigation.svelte.js'
  import { config } from '$stores/config.svelte.js'
  import { agentTabs } from '$stores/agentTabs.svelte.js'

  let showOverlay = $state(false)
  let showCommandPalette = $state(false)
  let showDocs = $state(false)

  const lowerTabShortcuts = [
    ['Ctrl+Shift+1', 0],
    ['Ctrl+Shift+2', 1],
    ['Ctrl+Shift+3', 2],
    ['Ctrl+Shift+4', 3],
    ['Ctrl+Shift+5', 4],
    ['Ctrl+Shift+6', 5],
    ['Ctrl+Shift+7', 6],
    ['Ctrl+Shift+8', 7],
    ['Ctrl+Shift+9', 8],
    ['Ctrl+Shift+0', 9],
  ]
  const paneFocusShortcuts = [
    ['Ctrl+Alt+[', 'left', 'Focus left lower pane'],
    ['Ctrl+Alt+]', 'right', 'Focus right lower pane'],
  ]
  const paneMoveShortcuts = [
    ['Ctrl+Alt+Shift+[', 'left', 'Move focused lower tab to left pane'],
    ['Ctrl+Alt+Shift+]', 'right', 'Move focused lower tab to right pane'],
  ]

  const activeShortcuts = $derived(getActiveShortcuts())
  const globalShortcuts = $derived(activeShortcuts.filter(s => s.category === 'global'))
  const agentViewShortcuts = $derived(activeShortcuts.filter(s => s.category === 'agents'))

  function switchLowerTab(index) {
    if (navigation.activeView !== 'agents') return
    agentTabs.selectFocusedTabByIndex(index)
  }

  function focusLowerPane(paneId) {
    if (navigation.activeView !== 'agents') return
    agentTabs.setFocusPane(paneId)
  }

  function moveFocusedLowerTab(paneId) {
    if (navigation.activeView !== 'agents') return
    agentTabs.moveFocusedTabToPane(paneId)
  }

  onMount(() => {
    register('Ctrl+K', () => { showCommandPalette = true }, 'Open command palette', 'global')
    register('Ctrl+,', () => navigation.setView('settings'), 'Open Settings', 'global')
    register('Ctrl+1', () => navigation.setView('agents'), 'Switch to Agents view', 'global')
    register('Ctrl+2', () => navigation.setView('automation'), 'Switch to Automation view', 'global')
    register('Ctrl+3', () => navigation.setView('server'), 'Switch to Server view', 'global')
    register('Ctrl+4', () => navigation.setView('settings'), 'Switch to Settings view', 'global')
    register('Ctrl+Shift+P', () => { showCommandPalette = true }, 'Alias for command palette', 'global')
    register('Ctrl+Shift+/', () => { showOverlay = true }, 'Show keyboard shortcuts', 'global')
    register('F1', () => { showDocs = true }, 'Open docs', 'global')
    register('Ctrl+=', () => config.zoomIn(), 'Zoom in', 'global')
    register('Ctrl+Shift+=', () => config.zoomIn(), 'Zoom in', 'global')
    register('Ctrl+-', () => config.zoomOut(), 'Zoom out', 'global')
    register('Ctrl+Shift+-', () => config.zoomOut(), 'Zoom out', 'global')
    register('Ctrl+0', () => config.zoomReset(), 'Reset zoom', 'global')
    for (const [shortcut, index] of lowerTabShortcuts) {
      const position = index === 9 ? 10 : index + 1
      register(shortcut, () => switchLowerTab(index), `Switch to lower tab ${position}`, 'agents')
    }
    for (const [shortcut, paneId, label] of paneFocusShortcuts) {
      register(shortcut, () => focusLowerPane(paneId), label, 'agents')
    }
    for (const [shortcut, paneId, label] of paneMoveShortcuts) {
      register(shortcut, () => moveFocusedLowerTab(paneId), label, 'agents')
    }

    const handler = (e) => handleKeydown(e)
    window.addEventListener('keydown', handler)
    return () => {
      window.removeEventListener('keydown', handler)
      unregister('Ctrl+K')
      unregister('Ctrl+,')
      unregister('Ctrl+1')
      unregister('Ctrl+2')
      unregister('Ctrl+3')
      unregister('Ctrl+4')
      unregister('Ctrl+Shift+P')
      unregister('Ctrl+Shift+/')
      unregister('F1')
      unregister('Ctrl+=')
      unregister('Ctrl+Shift+=')
      unregister('Ctrl+-')
      unregister('Ctrl+Shift+-')
      unregister('Ctrl+0')
      for (const [shortcut] of lowerTabShortcuts) unregister(shortcut)
      for (const [shortcut] of paneFocusShortcuts) unregister(shortcut)
      for (const [shortcut] of paneMoveShortcuts) unregister(shortcut)
    }
  })
</script>

<CommandPalette bind:open={showCommandPalette} />
<DocsModal bind:open={showDocs} onclose={() => showDocs = false} />

{#if showOverlay}
  <Modal bind:open={showOverlay} title="Keyboard Shortcuts" size="md" onclose={() => showOverlay = false}>
    <div class="space-y-4">
      <div>
        <div class="text-xs uppercase tracking-wider text-fg-muted mb-2">Global</div>
        <div class="space-y-3">
          {#each globalShortcuts as s}
            <div class="flex items-center justify-between">
              <span class="text-sm text-fg">{s.label}</span>
              <Kbd key={s.key} />
            </div>
          {/each}
        </div>
      </div>
      {#if agentViewShortcuts.length > 0}
        <div>
          <div class="text-xs uppercase tracking-wider text-fg-muted mb-2">Agents view</div>
          <div class="space-y-3">
            {#each agentViewShortcuts as s}
              <div class="flex items-center justify-between">
                <span class="text-sm text-fg">{s.label}</span>
                <Kbd key={s.key} />
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  </Modal>
{/if}
