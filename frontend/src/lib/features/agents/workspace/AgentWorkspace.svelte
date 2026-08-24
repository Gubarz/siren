<script>
  import { onMount } from 'svelte'
  import { GuiActionGroups } from '../../palette/GuiActions.js'
  import { registerCommandActions } from '../../palette/registry.js'
  import { GetCommandCatalog } from '../../../api/console.js'
  import { agentTabs } from '$stores/agentTabs.svelte.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'
  
  import AgentTopPane from './AgentTopPane.svelte'
  import AgentBottomPane from './AgentBottomPane.svelte'
  import AgentGlobalMenu from './AgentGlobalMenu.svelte'
  import SplitPane from '$components/patterns/SplitPane.svelte'
  import CommandFormV2 from '../modals/CommandFormV2.svelte'
  import ReconfigureAgentModal from '../modals/ReconfigureAgentModal.svelte'
  import { catalogToCategories } from './catalog.js'

  import { modalFor } from '../modals/registry.js'
  import { config } from '$stores/config.svelte.js'
  import { workspaceState } from '$stores/workspaceState.svelte.js'
  import { dispatchCommand } from '$stores/console.svelte.js'
  import { commandModal } from '$stores/ui/commandModal.svelte.js'
  import { selection } from '$stores/ui/selection.svelte.js'
  import { Modal } from '$stores/ui/Modal.svelte.js'

  let serverCategories = $state([])
  let agentCategories = $state([])
  let categories = $derived([...GuiActionGroups, ...agentCategories])

  const reconfigure = new Modal()

  let command = $derived(commandModal.command)
  let ActiveModal = $derived(command ? (modalFor(command.name) || CommandFormV2) : null)

  let isSessionBound = $derived(commandModal.useSession && commandModal.targetIDs.length > 0)
  let combinedSessionIDs = $derived(isSessionBound ? commandModal.targetIDs.join(', ') : '')
  let firstSessionID = $derived(isSessionBound ? commandModal.targetIDs[0] : '')

  onMount(async () => {
    try {
      const [sessionCatalog, serverCatalog] = await Promise.all([
        GetCommandCatalog('session'),
        GetCommandCatalog('server'),
      ])
      // Agent menus only get session commands — server commands (e.g. the
      // Sliver group) belong in the palette, not on the agents dropdown.
      agentCategories = catalogToCategories(sessionCatalog)
      const merged = new Map()
      for (const catalog of [sessionCatalog, serverCatalog]) {
        for (const group of catalog?.groups ?? []) {
          const key = group.title || group.id || 'Other'
          if (!merged.has(key)) merged.set(key, { category: key, commands: [] })
          merged.get(key).commands.push(...(group.commands ?? []))
        }
      }
      serverCategories = [...merged.values()]
      const paletteActions = []
      for (const cat of serverCategories) {
        for (const cmd of cat.commands) {
          paletteActions.push({
            id: `cmd-${cmd.name}`,
            label: `Run: ${cmd.name}`,
            description: cmd.description || `${cat.category} command`,
            icon: 'command',
            section: `Commands - ${cat.category}`,
            tags: [cmd.name, cat.category],
            on: () => commandModal.open({
              command: cmd,
              useSession: true,
              targetIDs: [...selection.agents],
            }),
          })
        }
      }
      registerCommandActions(paletteActions)
    } catch (error) {
      await dialog.alert(`Could not load Sliver commands: ${error}`)
    }
  })

  function executeSliverCommand(cmd) {
    if (isSessionBound) {
      for (const id of commandModal.targetIDs) {
        if (cmd === 'shell' || cmd.startsWith('shell ')) {
          agentTabs.launchShell(id, cmd.slice(5).trim())
        } else {
          agentTabs.openTab(id, 'console')
          dispatchCommand(id, cmd)
        }
      }
    } else {
      dispatchCommand('', cmd)
    }
    commandModal.close()
  }
</script>

<AgentGlobalMenu {categories} onReconfigure={(a) => reconfigure.show(a)} />

<SplitPane
  orientation="vertical"
  bind:size={() => workspaceState.topPaneHeight ?? config.topPaneHeight, (v) => { workspaceState.set('topPaneHeight', v); config.set('topPaneHeight', v) }}
  minSize={15}
  maxSize={85}
>
  {#snippet left()}
    <AgentTopPane />
  {/snippet}
  {#snippet right()}
    <AgentBottomPane />
  {/snippet}
</SplitPane>

{#if ActiveModal}
  <ActiveModal
    sessionID={combinedSessionIDs}
    {firstSessionID}
    {command}
    open={true}
    initialValues={commandModal.initialValues}
    onexecute={({ cmd }) => executeSliverCommand(cmd)}
    onclose={() => commandModal.close()}
  />
{/if}

<ReconfigureAgentModal bind:open={reconfigure.open} agent={reconfigure.data} />