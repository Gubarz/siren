<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import FileBrowser from '../../FileBrowser.svelte'
  import Button from '../../../../components/ui/Button.svelte'
  import IconButton from '../../../../components/ui/IconButton.svelte'
  import TextInput from '../../../../components/ui/TextInput.svelte'

  let {
    value = $bindable(''),
    sessionID = '',
    label = 'Remote path',
    description = '',
    placeholder = '/path/on/target',
    // 'file' = pick an existing file on the target filesystem
    // 'dir'  = pick a directory (destination path)
    mode = 'file',
  } = $props()

  let pickerOpen = $state(false)

  function handlePick({ path }) {
    if (!path) return
    value = path
    pickerOpen = false
  }

  function handleClear() {
    value = ''
  }
</script>

<div class="mb-2">
  <label class="mb-1 block text-sm font-medium text-fg" for="remote-file-picker-value">{label}</label>
  <div class="flex items-center gap-2">
    <div class="min-w-0 flex-1">
      <TextInput
        id="remote-file-picker-value"
        size="sm"
        {placeholder}
        bind:value
        spellcheck="false"
        autocomplete="off"
        class="font-mono"
      />
    </div>
    <Button
      color="dark"
      size="sm"
      icon={mode === 'dir' ? 'folder' : 'search'}
      onclick={() => pickerOpen = true}
      title={mode === 'dir' ? 'Browse target for a folder' : 'Browse target for a file'}
      disabled={!sessionID}
    >
      Browse target
    </Button>
    {#if value}
      <IconButton icon="x" label="Clear" tooltip="Clear" onclick={handleClear} />
    {/if}
  </div>
  {#if description}
    <span class="mt-1 block text-xs text-fg-muted">{description}</span>
  {/if}
</div>

{#if pickerOpen}
  <Modal bind:open={pickerOpen} title={mode === 'dir' ? 'Select folder on target' : 'Select file on target'} size="5xl">
    <div class="flex h-vh-60 flex-col">
      <FileBrowser
        sessionID={sessionID}
        picker={mode}
        startPath={value || ''}
        onpick={handlePick}
      />
    </div>
  </Modal>
{/if}
