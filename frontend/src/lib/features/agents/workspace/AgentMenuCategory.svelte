<script>
  import Button from '$components/ui/Button.svelte'
  import Menu from '$components/ui/Menu.svelte'
  import MenuItem from '$components/ui/MenuItem.svelte'
  
  import { overlays } from '$stores/ui/overlays.svelte.js'
  import { navigation } from '$stores/ui/navigation.svelte.js'
  import { config } from '$stores/config.svelte.js'
  import { commandModal } from '$stores/ui/commandModal.svelte.js'
  import { runGuiAction } from '../../palette/GuiActions.js'

  let { category, selectedAgents } = $props()

  let isOpen = $state(false)

  function isDisabled(cmd) {
    return !cmd.guiAction && selectedAgents.size === 0
  }

  function executeCommand(command) {
    if (isDisabled(command)) return
    
    isOpen = false
    
    if (command.guiAction) {
      runGuiAction(command.guiAction, { navigation, overlays, config })
    } else {
      commandModal.open({ command, useSession: true, targetIDs: [...selectedAgents] })
    }
  }
</script>

<div class="relative">
  <Button
    color="alternative"
    size="xs"
    class="border-0! bg-transparent! p-0! focus:ring-0! focus:outline-none! shadow-none! text-fg-muted hover:text-fg!"
  >
    {category.category}
  </Button>
  
  <Menu bind:isOpen minWidth="14rem">
    {#each category.commands as cmd}
      <MenuItem onclick={() => executeCommand(cmd)} disabled={isDisabled(cmd)}>
        {cmd.name}
      </MenuItem>
    {/each}
  </Menu>
</div>