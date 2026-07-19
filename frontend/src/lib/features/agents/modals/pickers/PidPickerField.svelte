<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import ProcessExplorer from '../../ProcessExplorer.svelte'
  import Button from '../../../../components/ui/Button.svelte'
  import IconButton from '../../../../components/ui/IconButton.svelte'
  import TextInput from '../../../../components/ui/TextInput.svelte'

  let {
    value = $bindable(0),
    sessionID = '',
    label = 'Process PID',
    processName = $bindable(''),
  } = $props()

  let pickerOpen = $state(false)
  let pidText = $state(value ? String(value) : '')

  function handlePick({ pid, name }) {
    value = pid
    pidText = String(pid)
    processName = name || ''
    pickerOpen = false
  }

  function handleClear() {
    value = 0
    pidText = ''
    processName = ''
  }
</script>

<div class="mb-2">
  <label class="mb-1 block text-sm font-medium text-fg" for="pid-picker-value">{label}</label>
  <div class="flex items-center gap-2">
    <div class="w-24">
      <TextInput
        id="pid-picker-value"
        type="number"
        size="sm"
        placeholder="PID"
        bind:value={pidText}
        oninput={(e) => { value = parseInt(e.target.value, 10) || 0; processName = '' }}
      />
    </div>
    <Button color="dark" size="sm" icon="search" onclick={() => pickerOpen = true} title="Pick from process list">
      Pick
    </Button>
    {#if pidText}
      <IconButton icon="x" label="Clear" tooltip="Clear" onclick={handleClear} />
    {/if}
  </div>
  {#if processName}
    <span class="mt-1 block text-xs text-fg-muted">{processName}</span>
  {/if}
</div>

{#if pickerOpen}
  <Modal bind:open={pickerOpen} title="Select Process" size="5xl">
    <div class="max-h-vh-60 overflow-y-auto">
      <ProcessExplorer sessionID={sessionID} picker={true} onpick={handlePick} />
    </div>
  </Modal>
{/if}
