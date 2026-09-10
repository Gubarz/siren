<script>
  import { selection } from '$stores/ui/selection.svelte.js'
  import Button from '$components/ui/Button.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import Menu from '$components/ui/Menu.svelte'
  import MenuItem from '$components/ui/MenuItem.svelte'
  import { overlays } from '$stores/ui/overlays.svelte.js'
  import { navigation } from '$stores/ui/navigation.svelte.js'
  import { config } from '$stores/config.svelte.js'
  import { commandModal } from '$stores/ui/commandModal.svelte.js'
  import { contextMenu } from '$stores/ui/contextMenu.svelte.js'
  import { agentTabs } from '$stores/agentTabs.svelte.js'
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js'
  import { tagsModal } from '$stores/ui/tagsModal.svelte.js'
  import { addToCase } from '$stores/ui/addToCase.svelte.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'
  import { agentColors } from '$stores/resources/agentColors.svelte.js'
  import { sessions } from '$stores/resources/sessions.svelte.js'
  import { beacons } from '$stores/resources/beacons.svelte.js'
  import { discoveries } from '$stores/resources/discoveries.svelte.js'
  import { knownAgents } from '$stores/knownAgents.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'
  import { SetAgentColor } from '../../../api/tags.js'
  import { errorMessage } from '../../../utils/errors.js'
  import { runGuiAction } from '../../palette/GuiActions.js'
  import { ROW_COLORS, colorHex } from '../../../utils/agentColors.js'
  import { Modal } from '$stores/ui/Modal.svelte.js'
  import BeaconDetailModal from '../modals/BeaconDetailModal.svelte'
  import { createAgentActions } from './agentActions.js'
  import { buildAgentActionsSections, buildAgentContextSections, buildCommandCategories } from './agentContextActions.js'
  import { buildAgentContextMenuHandlers } from './agentMenuHandlers.js'
  import { mergeLostAgents } from './useAgentData.svelte.js'

  useResource(sessions, beacons, agentColors)

  let { categories = [], onReconfigure = () => {} } = $props()

  const beaconDetail = new Modal()

  let workspaceOpen = $state(false)
  let guiCommands = $derived((categories.find((category) => category.category === 'GUI')?.commands || []).filter(isAgentWorkspaceCommand))
  let commandCategories = $derived(categories.filter((category) => category.category !== 'GUI' && category.category !== 'Generic'))

  // The global menus resolve selection against the same live+known merge the
  // table renders, so a lost-only selection yields lost rows (not null) and
  // the context builder can hand back the reduced history menu.
  let knownAgentData = $derived(knownAgents.data)
  let combinedData = $derived.by(() => {
    const live = [
      ...(beacons.data ?? []).map((b) => ({ ...b, _kind: 'beacon' })),
      ...(sessions.data ?? []).map((s) => ({ ...s, _kind: 'session' })),
    ]
    return [...live, ...mergeLostAgents(live, knownAgentData)]
  })
  let agentMap = $derived.by(() => new Map(combinedData.map((agent) => [agent.ID, agent])))
  let selectedAgentRows = $derived(
    [...selection.agents].map((id) => agentMap.get(id)).filter(Boolean),
  )
  let liveSelectedRows = $derived(selectedAgentRows.filter((agent) => !agent._lost))
  let contextAgent = $derived(selectedAgentRows[0] ?? null)

  const actions = createAgentActions({
    dialog,
    discoveries,
    agentTabs,
    // Discovery targets are live rows only; the anchor is appended so a
    // lost-only selection still resolves to itself for the history menu.
    selectedAgentIDsIncluding: (agent) => [
      ...liveSelectedRows.filter((row) => row.ID !== agent.ID).map((row) => row.ID),
      agent.ID,
    ],
  })
  const { newShell, renameAgent, promoteBeacon, demoteSession } = actions

  function getAgentLabel(id) {
    const a = agentMap.get(id)
    return a?.Name || a?.Hostname || id
  }

  function renameSelected() {
    const agent = liveSelectedRows[0]
    if (agent) void renameAgent(agent)
  }

  function openReconfigureSelected() {
    const agent = liveSelectedRows[0]
    if (agent) onReconfigure(agent)
  }

  function openTagsForSelected() {
    const agent = selectedAgentRows[0]
    if (agent) tagsModal.openTags('agent', agent.ID, getAgentLabel(agent.ID))
  }

  function openCommentsForSelected() {
    const agent = selectedAgentRows[0]
    if (agent) commentsModal.openComments('agent', agent.ID, getAgentLabel(agent.ID))
  }

  function openAddToCaseForSelected() {
    const agent = selectedAgentRows[0]
    if (agent) addToCase.open({ collection: 'agent', itemID: agent.ID, label: getAgentLabel(agent.ID) })
  }

  async function setColorForSelected(color) {
    if (selection.agents.size === 0) return
    try {
      await Promise.all([...selection.agents].map((id) => SetAgentColor(id, color)))
      await agentColors.refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Color failed: '), 'Row Color')
    }
  }

  async function setAgentRowColor(agents, color) {
    try {
      await Promise.all(agents.map((agent) => SetAgentColor(agent.ID, color)))
      await agentColors.refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Color failed: '), 'Row Color')
    }
  }

  function isAgentWorkspaceCommand(command) {
    const action = command.guiAction
    if (!action) return false
    if (action.serverTab) return false
    return !action.view || action.view === 'agents'
  }

  function executeCommand(command, targetIDs = []) {
    workspaceOpen = false
    if (command.guiAction) {
      runGuiAction(command.guiAction, { navigation, overlays, config })
      return
    }
    if (targetIDs.length === 0) return
    commandModal.open({ command, useSession: true, targetIDs })
  }

  function openActionsMenu(event) {
    event.stopPropagation()
    workspaceOpen = false
    const rect = event.currentTarget.getBoundingClientRect()
    const agent = contextAgent
    const isBeacon = agent?._kind === 'beacon'
    const isWindows = (agent?.OS || '').toLowerCase() === 'windows'
    const hasInteractiveSession = isBeacon && (sessions.data ?? []).some((s) => s.Name === agent?.Name)

    // Empty selection keeps the old disabled core menu; lost/mixed selections
    // go through the context builder so lost rows get history actions and
    // live actions only ever target live agents.
    if (selectedAgentRows.length === 0) {
      contextMenu.open({
        x: rect.left,
        y: rect.bottom + 4,
        sections: buildAgentActionsSections({
          agent: null,
          isBeacon: false,
          isWindows: false,
          hasInteractiveSession: false,
          targetAgents: [],
          agentTabs,
          promoteBeacon,
          demoteSession,
          newShell,
          findAttackPaths: (agents) => {
            for (const target of agents) agentTabs.openTab(target.ID, 'bloodhound')
          },
        }),
      })
      return
    }

    contextMenu.open({
      x: rect.left,
      y: rect.bottom + 4,
      sections: buildAgentContextSections({
        agent,
        isBeacon,
        isWindows,
        hasInteractiveSession,
        catalog: liveSelectedRows.length > 0 ? commandCategories : [],
        targetIDs: [...selection.agents],
        targetAgents: selectedAgentRows,
        agentTabs,
        automationRules: [],
        contextMenuHandlers: buildAgentContextMenuHandlers({
          actions, agentTabs, addToCase, beaconDetail, dialog, knownAgents, selection,
          overrides: {
            openReconfigure: (agent) => onReconfigure(agent),
            openTags: (type, id, label) => tagsModal.openTags(type, id, label),
            openComments: (type, id, label) => commentsModal.openComments(type, id, label),
            setAgentRowColor,
            executeAgentCommand: (command, targetIDs) => commandModal.open({ command, useSession: true, targetIDs }),
          },
        }),
      }),
    })
  }

  function openCommandsMenu(event) {
    event.stopPropagation()
    workspaceOpen = false
    const rect = event.currentTarget.getBoundingClientRect()
    const liveTargetIDs = liveSelectedRows.map((agent) => agent.ID)
    let items
    if (commandCategories.length === 0) {
      items = [{ label: 'Loading commands...', disabled: true }]
    } else if (liveTargetIDs.length === 0) {
      items = [{ label: 'No live agents selected', disabled: true }]
    } else {
      items = buildCommandCategories({
        catalog: commandCategories,
        targetIDs: liveTargetIDs,
        executeAgentCommand: executeCommand,
      })
    }

    contextMenu.open({
      x: rect.left,
      y: rect.bottom + 4,
      sections: [{ items }],
    })
  }

  function openManageMenu(event) {
    event.stopPropagation()
    workspaceOpen = false
    const rect = event.currentTarget.getBoundingClientRect()
    const noSelection = selectedAgentRows.length === 0
    const noLiveSelection = liveSelectedRows.length === 0
    const paletteItems = ROW_COLORS.map((name) => ({
      label: name[0].toUpperCase() + name.slice(1),
      color: colorHex(name),
      on: () => setColorForSelected(name),
    }))
    contextMenu.open({
      x: rect.left,
      y: rect.bottom + 4,
      sections: [
        {
          items: [
            { icon: 'pen', label: 'Rename Agent\u2026', disabled: noLiveSelection, on: renameSelected },
            { icon: 'sliders', label: 'Reconfigure\u2026', disabled: noLiveSelection, on: openReconfigureSelected },
            { icon: 'tag', label: 'Tags / Color\u2026', disabled: noSelection, on: openTagsForSelected },
            { icon: 'message-square', label: 'Comments / Notes\u2026', disabled: noSelection, on: openCommentsForSelected },
            { icon: 'folder-plus', label: 'Add to case\u2026', disabled: noSelection, on: openAddToCaseForSelected },
          ],
        },
        { palette: true, items: paletteItems, clearItem: { label: 'Clear', on: () => setColorForSelected('') } },
      ],
    })
  }
</script>

<div class="flex bg-chrome-header px-3 py-2 border-b border-line gap-2 text-sm">
  <div class="relative">
    <Button
      color="alternative"
      size="xs"
      class="border-0! bg-transparent! shadow-none! text-fg-muted hover:text-fg! focus:ring-0! focus:outline-none!"
      aria-haspopup="true"
      aria-expanded={workspaceOpen}
    >
      <Icon name="panel-left" size={14} />
      Workspace
      <Icon name="chevron-down" size={12} />
    </Button>

    <Menu bind:isOpen={workspaceOpen} minWidth="15rem">
      {#each guiCommands as cmd}
        <MenuItem onclick={() => executeCommand(cmd)}>
          {#if cmd.guiAction?.icon}
            <Icon name={cmd.guiAction.icon} size={14} />
          {/if}
          <span>{cmd.name}</span>
        </MenuItem>
      {/each}
    </Menu>
  </div>

  <div class="relative">
    <Button
      color="alternative"
      size="xs"
      class="border-0! bg-transparent! shadow-none! text-fg-muted hover:text-fg! focus:ring-0! focus:outline-none!"
      aria-haspopup="true"
      onclick={openActionsMenu}
    >
      <Icon name="terminal" size={14} />
      Actions
      <Icon name="chevron-down" size={12} />
    </Button>
  </div>

  <div class="relative">
    <Button
      color="alternative"
      size="xs"
      class="border-0! bg-transparent! shadow-none! text-fg-muted hover:text-fg! focus:ring-0! focus:outline-none!"
      aria-haspopup="true"
      onclick={openCommandsMenu}
    >
      <Icon name="command" size={14} />
      Commands
      <Icon name="chevron-down" size={12} />
    </Button>
  </div>

  <div class="relative">
    <Button
      color="alternative"
      size="xs"
      class="border-0! bg-transparent! shadow-none! text-fg-muted hover:text-fg! focus:ring-0! focus:outline-none!"
      aria-haspopup="true"
      onclick={openManageMenu}
    >
      <Icon name="sliders" size={14} />
      Manage
      <Icon name="chevron-down" size={12} />
    </Button>
  </div>
</div>

<BeaconDetailModal bind:open={beaconDetail.open} beaconID={beaconDetail.data} />
