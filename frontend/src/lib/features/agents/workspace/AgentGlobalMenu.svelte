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
  import { TAB_META } from '$stores/agentTabs.svelte.js'
  import { sessions } from '$stores/resources/sessions.svelte.js'
  import { beacons } from '$stores/resources/beacons.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'
  import { SetAgentColor } from '../../../api/tags.js'
  import { RenameAgent } from '../../../api/agents.js'
  import { errorMessage } from '../../../utils/errors.js'
  import { runGuiAction } from '../../palette/GuiActions.js'
  import { ROW_COLORS, colorHex } from '../../../utils/agentColors.js'

  useResource(sessions, beacons, agentColors)

  let { categories = [], onReconfigure = () => {} } = $props()

  let selectedAgents = $derived(selection.agents)
  let workspaceOpen = $state(false)
  let guiCommands = $derived((categories.find((category) => category.category === 'GUI')?.commands || []).filter(isAgentWorkspaceCommand))
  let commandCategories = $derived(categories.filter((category) => category.category !== 'GUI' && category.category !== 'Generic'))

  let agentMap = $derived(() => {
    const map = new Map()
    for (const s of (sessions.data || [])) map.set(s.ID, s)
    for (const b of (beacons.data || [])) map.set(b.ID, b)
    return map
  })

  function getAgentLabel(id) {
    const a = agentMap.get(id)
    return a?.Name || a?.Hostname || id
  }

  function openTabForSelected(type) {
    for (const id of selectedAgents) agentTabs.openTab(id, type)
  }

  function tabItem(type) {
    const meta = TAB_META[type]
    const disabled = selectedAgents.size === 0
    return { icon: meta?.icon ?? 'info', label: meta?.label ?? type, disabled, on: () => openTabForSelected(type) }
  }

  function newShellForSelected() {
    for (const id of selectedAgents) agentTabs.launchShell(id, '')
  }

  async function renameSelected() {
    if (selectedAgents.size === 0) return
    const id = [...selectedAgents][0]
    const label = getAgentLabel(id)
    const name = await dialog.prompt('New name:', 'Rename Agent', label)
    if (!name || name === label) return
    try {
      await RenameAgent(id, name)
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Rename failed: '), 'Rename Agent')
    }
  }

  function openReconfigureSelected() {
    if (selectedAgents.size === 0) return
    const id = [...selectedAgents][0]
    const agent = agentMap.get(id)
    if (agent) onReconfigure(agent)
  }

  function openTagsForSelected() {
    if (selectedAgents.size === 0) return
    const id = [...selectedAgents][0]
    tagsModal.openTags('agent', id, getAgentLabel(id))
  }

  function openCommentsForSelected() {
    if (selectedAgents.size === 0) return
    const id = [...selectedAgents][0]
    commentsModal.openComments('agent', id, getAgentLabel(id))
  }

  function openAddToCaseForSelected() {
    if (selectedAgents.size === 0) return
    const id = [...selectedAgents][0]
    addToCase.open({ collection: 'agent', itemID: id, label: getAgentLabel(id) })
  }

  async function setColorForSelected(color) {
    if (selectedAgents.size === 0) return
    try {
      await Promise.all([...selectedAgents].map(id => SetAgentColor(id, color)))
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

  function isDisabled(cmd) {
    return !cmd.guiAction && selectedAgents.size === 0
  }

  function executeCommand(command) {
    if (isDisabled(command)) return
    workspaceOpen = false
    if (command.guiAction) {
      runGuiAction(command.guiAction, { navigation, overlays, config })
    } else {
      commandModal.open({ command, useSession: true, targetIDs: [...selectedAgents] })
    }
  }

  function openActionsMenu(event) {
    event.stopPropagation()
    workspaceOpen = false
    const rect = event.currentTarget.getBoundingClientRect()
    const disabled = selectedAgents.size === 0
    contextMenu.open({
      x: rect.left,
      y: rect.bottom + 4,
      sections: [{
        items: [
          tabItem('console'),
          { icon: 'terminal-square', label: 'New Shell', disabled, on: newShellForSelected },
          tabItem('fileBrowser'),
          tabItem('tunneling'),
          tabItem('processExplorer'),
          tabItem('services'),
          {
            icon: 'ellipsis-vertical', label: 'More',
            children: [
              tabItem('screenshot'),
              tabItem('grep'),
              tabItem('env'),
              tabItem('registryBrowser'),
              tabItem('netstat'),
              tabItem('ifconfig'),
              tabItem('privileges'),
            ],
          },
        ],
      }],
    })
  }

  function openCommandsMenu(event) {
    event.stopPropagation()
    workspaceOpen = false
    const rect = event.currentTarget.getBoundingClientRect()
    const items = commandCategories.length === 0
      ? [{ label: 'Loading commands...', disabled: true }]
      : commandCategories.map((category) => ({
          icon: 'command',
          label: category.category,
          children: category.commands.map((cmd) => ({
            label: cmd.name,
            description: cmd.unavailable || cmd.description || '',
            disabled: isDisabled(cmd),
            on: () => executeCommand(cmd),
          })),
        }))

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
    const disabled = selectedAgents.size === 0
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
            { icon: 'pen', label: 'Rename Agent\u2026', disabled, on: renameSelected },
            { icon: 'sliders', label: 'Reconfigure\u2026', disabled, on: openReconfigureSelected },
            { icon: 'tag', label: 'Tags / Color\u2026', disabled, on: openTagsForSelected },
            { icon: 'message-square', label: 'Comments / Notes\u2026', disabled, on: openCommentsForSelected },
            { icon: 'folder-plus', label: 'Add to case\u2026', disabled, on: openAddToCaseForSelected },
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
