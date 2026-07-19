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
  import { runGuiAction } from '../../palette/GuiActions.js'

  let { categories = [] } = $props()

  let selectedAgents = $derived(selection.agents)
  let workspaceOpen = $state(false)
  let guiCommands = $derived((categories.find((category) => category.category === 'GUI')?.commands || []).filter(isAgentWorkspaceCommand))
  let commandCategories = $derived(categories.filter((category) => category.category !== 'GUI'))
  let commandCount = $derived(commandCategories.reduce((total, category) => total + category.commands.length, 0))

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
</script>

<div class="flex bg-chrome-header px-3 py-2 border-b border-line gap-2 text-sm">
  <div class="relative">
    <Button
      color="alternative"
      size="xs"
      class="border-0! bg-transparent! shadow-none! text-fg-muted hover:text-fg!"
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
      class="border-0! bg-transparent! shadow-none! text-fg-muted hover:text-fg!"
      aria-haspopup="true"
      onclick={openCommandsMenu}
    >
      <Icon name="command" size={14} />
      Commands
      <span class="rounded bg-row-hover px-1 text-2xs text-fg-muted">{commandCount}</span>
      <Icon name="chevron-down" size={12} />
    </Button>
  </div>
</div>
