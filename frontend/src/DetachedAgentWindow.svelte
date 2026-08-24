<script>
  import { onMount } from 'svelte'
  import AppShell from '$components/layout/AppShell.svelte'
  import GlobalDialog from '$components/system/GlobalDialog.svelte'
  import Toasts from '$components/system/Toasts.svelte'
  import ContextMenuRoot from '$components/system/ContextMenuRoot.svelte'
  import PanelRouter from '$components/system/PanelRouter.svelte'
  import AddToCaseRoot from '$components/system/AddToCaseRoot.svelte'
  import EntityCommentsRoot from '$components/system/EntityCommentsRoot.svelte'
  import EntityTagsRoot from '$components/system/EntityTagsRoot.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import { GetDetachedAgentTab, ReattachAgentTab } from '$api/detachedTabs.js'
  import { closeWindow, minimizeWindow, toggleMaximizeWindow, setWindowZoom } from '$api/runtime.js'
  import { dispatchCommand } from '$stores/console.svelte.js'
  import { config } from '$stores/config.svelte.js'
  import { agentTabs } from '$stores/agentTabs.svelte.js'
  import { applyThemePreference, watchSystemThemePreference } from '$stores/ui/theme.svelte.js'
  import { errorMessage } from '$utils/errors.js'
  import { resolveTab } from '$features/agents/workspace/tabRegistry.js'

  let { token = '' } = $props()
  let envelope = $state(null)
  let loadError = $state('')
  let reattaching = $state(false)

  let tab = $derived(envelope?.tab || null)
  let shellMap = $derived(envelope?.shell?.id ? { [envelope.shell.id]: envelope.shell } : {})
  let resolved = $derived(tab ? resolveTab(tab, {
    shellsByID: shellMap,
    isActive: () => true,
    openShell: (sessionID, raw = '') => agentTabs.launchShell(sessionID, raw),
    runConsoleCommand: (sessionID, command) => dispatchCommand(sessionID, command),
  }) : null)

  $effect(() => {
    const zoom = Number(config?.zoom)
    const appZoom = Number.isFinite(zoom) && zoom > 0 ? zoom : 1
    setWindowZoom(appZoom)
  })

  onMount(() => {
    applyThemePreference()
    const stopWatchingTheme = watchSystemThemePreference()
    void loadTab()
    return stopWatchingTheme
  })

  async function loadTab() {
    try {
      envelope = JSON.parse(await GetDetachedAgentTab(token))
      document.title = envelope?.tab?.label || 'siren agent tab'
    } catch (err) {
      loadError = errorMessage(err, 'Could not load detached tab: ')
    }
  }

  async function reattach() {
    if (reattaching) return
    reattaching = true
    try {
      await ReattachAgentTab(token)
    } catch (err) {
      loadError = errorMessage(err, 'Could not return tab to the main window: ')
      reattaching = false
    }
  }

  function handlePress(action) {
    return (event) => {
      if (event.button !== 0) return
      event.preventDefault()
      action()
    }
  }
</script>

<AppShell>
  <div
    class="relative z-titlebar h-10 shrink-0 select-none border-b border-line bg-chrome-header flex items-stretch"
    style="--wails-draggable:drag"
  >
    <div class="flex min-w-0 flex-1 items-center gap-2 px-3 text-sm text-fg-muted">
      <Icon name="external-link" size={14} />
      <span class="truncate">{tab?.label || 'Agent tab'}</span>
    </div>
    <button
      type="button"
      style="--wails-draggable:no-drag"
      class="h-full px-3 text-xs text-fg-muted transition-colors hover:bg-row-hover hover:text-fg disabled:opacity-50"
      disabled={!tab || reattaching}
      title="Return this tab to the main window"
      onclick={reattach}
    >
      {reattaching ? 'Returning…' : 'Return to main window'}
    </button>
    <button type="button" style="--wails-draggable:no-drag" class="w-12 h-full flex items-center justify-center text-fg-muted hover:bg-row-hover hover:text-fg" aria-label="Minimize window" onpointerdown={handlePress(minimizeWindow)}>
      <svg width="12" height="12" viewBox="0 0 12 12"><rect fill="currentColor" width="10" height="1" x="1" y="6"></rect></svg>
    </button>
    <button type="button" style="--wails-draggable:no-drag" class="w-12 h-full flex items-center justify-center text-fg-muted hover:bg-row-hover hover:text-fg" aria-label="Maximize window" onpointerdown={handlePress(toggleMaximizeWindow)}>
      <svg width="12" height="12" viewBox="0 0 12 12"><rect width="9" height="9" x="1.5" y="1.5" fill="none" stroke="currentColor"></rect></svg>
    </button>
    <button type="button" style="--wails-draggable:no-drag" class="w-12 h-full flex items-center justify-center text-fg-muted hover:bg-danger-500! hover:text-white!" aria-label="Close window" onpointerdown={handlePress(closeWindow)}>
      <svg width="12" height="12" viewBox="0 0 12 12"><polygon fill="currentColor" points="11 1.5 10.5 1 6 5.5 1.5 1 1 1.5 5.5 6 1 10.5 1.5 11 6 6.5 10.5 11 11 10.5 6.5 6"></polygon></svg>
    </button>
  </div>

  <main class="relative flex min-h-0 flex-1 flex-col overflow-hidden">
    {#if loadError}
      <div class="flex flex-1 items-center justify-center p-6 text-danger-500">{loadError}</div>
    {:else if resolved}
      {@const Component = resolved.component}
      <div class="flex h-full min-h-0 flex-col">
        <Component {...resolved.props} />
      </div>
    {:else}
      <div class="flex flex-1 items-center justify-center text-fg-muted">Loading agent tab…</div>
    {/if}
  </main>

  <GlobalDialog />
  <Toasts />
  <ContextMenuRoot />
  <PanelRouter />
  <AddToCaseRoot />
  <EntityCommentsRoot />
  <EntityTagsRoot />
</AppShell>
