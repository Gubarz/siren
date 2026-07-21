<script>
  import { onMount } from 'svelte'
  import { onSliverEvent } from './lib/api/runtime.js'
  import AppShell from '$components/layout/AppShell.svelte'
  import TitleBar from './lib/components/layout/TitleBar.svelte'
  import StatusBar from './lib/components/layout/StatusBar.svelte'
  import GlobalDialog from './lib/components/system/GlobalDialog.svelte'
  import Toasts from './lib/components/system/Toasts.svelte'
  import ContextMenuRoot from './lib/components/system/ContextMenuRoot.svelte'
  import PanelRouter from './lib/components/system/PanelRouter.svelte'
  import KeyboardShortcutsRoot from './lib/components/system/KeyboardShortcutsRoot.svelte'
  import AddToCaseRoot from './lib/components/system/AddToCaseRoot.svelte'
  import EntityCommentsRoot from './lib/components/system/EntityCommentsRoot.svelte'
  import EntityTagsRoot from './lib/components/system/EntityTagsRoot.svelte'
  import AgentWorkspace from './lib/features/agents/workspace/AgentWorkspace.svelte'
  import AutomationView from './lib/features/automation/AutomationView.svelte'
  import ConnectionScreen from './lib/features/connection/ConnectionScreen.svelte'
  import ServerView from './lib/features/server/ServerView.svelte'
  import SettingsView from './lib/features/settings/SettingsView.svelte'
  import { connection } from '$stores/connection.svelte.js'
  import { config } from '$stores/config.svelte.js'
  import { sessions } from '$stores/resources/sessions.svelte.js'
  import { beacons } from '$stores/resources/beacons.svelte.js'
  import { pushEvent } from '$stores/resources/events.svelte.js'
  import { navigation } from '$stores/ui/navigation.svelte.js'
  import { applyThemePreference, watchSystemThemePreference } from '$stores/ui/theme.svelte.js'
  import { installClientLogHandlers } from './lib/utils/clientLog.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(sessions, beacons)

  let activeView = $derived(navigation.activeView)
  let conn = $derived(connection || { connected: false, profile: '', version: '', reconnecting: false })
  let sessionData = $derived(sessions?.data || [])
  let beaconData = $derived(beacons?.data || [])

  // Seed from navigation once, then serverTab is the source of truth for
  // the visible sub-tab. Writing back to navigation is fire-and-forget so
  // palette actions calling `navigation.setServerTab(...)` from elsewhere
  // still work — the local $state just doesn't fight them.
  let serverTab = $state(navigation.serverTab)

  $effect(() => {
    if (serverTab) navigation.setServerTab(serverTab)
  })

  $effect(() => {
    const zoom = Number(config?.zoom)
    const appZoom = Number.isFinite(zoom) && zoom > 0 ? zoom : 1
    document.documentElement.style.fontSize = `${16 * appZoom}px`
  })

  onMount(() => {
    applyThemePreference()
    const stopWatchingTheme = watchSystemThemePreference()
    const stopClientLogHandlers = installClientLogHandlers()
    const stopSliverEvents = onSliverEvent((event) => {
      const type = event.type || ''
      if (type === 'stream-closed') {
        if (conn.connected) connection.startReconnecting(conn.profile)
        return
      }
      pushEvent(event)
    })

    return () => {
      stopWatchingTheme()
      stopClientLogHandlers()
      stopSliverEvents?.()
    }
  })

  function handleConnected(profile) {
    connection.markConnected(profile)
  }
</script>

<AppShell>
  <TitleBar />

  <main class="relative flex flex-col flex-1 overflow-hidden">
    {#if !conn.connected}
      <ConnectionScreen onconnected={handleConnected} />
    {:else}
      <div class="flex flex-1 min-h-0 flex-col overflow-hidden" class:hidden={activeView !== 'agents'}>
        <AgentWorkspace />
      </div>
      <div class="flex flex-1 min-h-0 flex-col overflow-hidden" class:hidden={activeView !== 'server'}>
        <ServerView bind:active={serverTab} />
      </div>
      {#if activeView !== 'agents' && activeView !== 'server'}
        <div class="flex flex-1 min-h-0 flex-col overflow-hidden">
          {#if activeView === 'automation'}
            <AutomationView />
          {:else if activeView === 'settings'}
            <SettingsView />
          {/if}
        </div>
      {/if}

      <StatusBar
        serverVersion={conn.version}
        sessionCount={sessionData.length}
        beaconCount={beaconData.length}
      />
    {/if}
  </main>

  <GlobalDialog />
  <Toasts />
  <ContextMenuRoot />
  <PanelRouter />
  <KeyboardShortcutsRoot />
  <AddToCaseRoot />
  <EntityCommentsRoot />
  <EntityTagsRoot />
</AppShell>
