<script>
  import { presets } from '../../utils/presets.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import Icon from '../ui/Icon.svelte'
  import Button from '../ui/Button.svelte'
  import IconButton from '../ui/IconButton.svelte'
  import TextInput from '../ui/TextInput.svelte'
  import Menu from '../ui/Menu.svelte'
  import MenuItem from '../ui/MenuItem.svelte'

  let {
    commandPath = '',
    currentValues = {},
    onapply,
  } = $props()

  let list = $state([])
  let dropOpen = $state(false)
  let saving = $state(false)
  let newName = $state('')

  $effect(() => {
    list = presets.list(commandPath)
  })

  function applyPreset(name) {
    const p = presets.get(commandPath, name)
    if (p) { onapply?.(p.values); dropOpen = false }
  }

  function confirmSave() {
    const name = newName.trim()
    if (!name) return
    presets.save(commandPath, name, currentValues)
    list = presets.list(commandPath)
    saving = false
    newName = ''
  }

  function cancelSave() {
    saving = false
    newName = ''
  }

  async function removePreset(name) {
    const ok = await dialog.confirm(`Delete preset "${name}"?`, 'Delete Preset')
    if (!ok) return
    presets.remove(commandPath, name)
    list = presets.list(commandPath)
  }
</script>

<div class="inline-flex items-center">
  {#if saving}
    <div class="flex items-center gap-1">
      <Icon name="bookmark" size={14} class="text-brand shrink-0" />
      <div class="w-36">
        <TextInput
          size="sm"
          bind:value={newName}
          placeholder="Preset name..."
          autofocus
          onkeydown={(e) => {
            if (e.key === 'Enter') confirmSave()
            if (e.key === 'Escape') cancelSave()
          }}
        />
      </div>
      <IconButton icon="check" label="Save" tooltip="Save" onclick={confirmSave} disabled={!newName.trim()} />
      <IconButton icon="x" label="Cancel" tooltip="Cancel" onclick={cancelSave} color="red" />
    </div>
  {:else}
    <Button
      color="alternative"
      size="xs"
      icon="bookmark"
      aria-haspopup="true"
      aria-expanded={dropOpen}
    >
      <span>Presets</span>
      {#if list.length > 0}
        <span class="inline-flex items-center justify-center min-w-4 h-4 px-1 rounded-full bg-brand text-white text-2xs font-semibold">{list.length}</span>
      {/if}
    </Button>
    <Menu bind:isOpen={dropOpen} minWidth="11rem">
      {#if list.length === 0}
        <div class="px-3 py-2 text-xs text-fg-muted">No saved presets</div>
      {:else}
        {#each list as p}
          <MenuItem
            onclick={() => applyPreset(p.name)}
            class="justify-between"
          >
            <span class="truncate">{p.name}</span>
            <IconButton
              icon="x"
              label="Delete preset"
              tooltip="Delete preset"
              color="red"
              onclick={(e) => { e.stopPropagation(); removePreset(p.name) }}
            />
          </MenuItem>
        {/each}
      {/if}
      <div class="h-px bg-line my-1"></div>
      <MenuItem
        variant="accent"
        onclick={() => { dropOpen = false; saving = true }}
      >
        <Icon name="plus" size={14} />
        Save current preset…
      </MenuItem>
    </Menu>
  {/if}
</div>
