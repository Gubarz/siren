<script>
  import Modal from '$components/patterns/Modal.svelte'
  import { quote } from '$utils/shell.js'
  import Button from '$components/ui/Button.svelte'
  import HostPortField from '$components/forms/HostPortField.svelte'
  import PresetPicker from '$components/forms/PresetPicker.svelte'

  let {
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
    commandPath = '',
    title = '',
    description = '',
    example = '',
    bindLabel = '',
    bindPlaceholder = '',
    bindPortPlaceholder = '',
    bindDescription = '',
    bindRequired = false,
    bindDefault = '',
    remoteLabel = '',
    remotePlaceholder = '',
    remotePortPlaceholder = '',
    remoteDescription = '',
    remoteRequired = false,
    remoteDefault = '',
  } = $props()

  let bind = $state('')
  let remote = $state('')

  $effect.pre(() => {
    bind = initialValues['bind'] || bindDefault
    remote = initialValues['remote'] || remoteDefault
  })

  let cmdPreview = $derived.by(() => {
    const parts = [commandPath]
    if (bind) parts.push('--bind', quote(bind))
    if (remote) parts.push('--remote', quote(remote))
    return parts.filter(Boolean).join(' ')
  })

  let isDisabled = $derived.by(() => {
    if (bindRequired && !bind) return true
    if (remoteRequired && !remote) return true
    return false
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title={title} size="xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">{@html description}</p>
  <p class="text-fg-muted text-xs mb-4 opacity-70">{@html example}</p>

  <div class="mb-3">
    <HostPortField
      bind:value={bind}
      label={bindLabel}
      hostPlaceholder={bindPlaceholder}
      portPlaceholder={bindPortPlaceholder}
      description={bindDescription}
      required={bindRequired}
    />
  </div>

  <div class="mb-3">
    <HostPortField
      bind:value={remote}
      label={remoteLabel}
      hostPlaceholder={remotePlaceholder}
      portPlaceholder={remotePortPlaceholder}
      description={remoteDescription}
      required={remoteRequired}
    />
  </div>

  <div class="mb-4">
    <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
    <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview}</code>
  </div>

  {#snippet footer()}
    <div class="flex justify-between items-center">
      <PresetPicker
        {commandPath}
        currentValues={{ bind, remote }}
        onapply={(values) => {
          if (values['bind'] != null) bind = values['bind']
          if (values['remote'] != null) remote = values['remote']
        }}
      />
      <div class="flex gap-2">
        <Button color="dark" onclick={() => open = false}>Cancel</Button>
        <Button color="primary" onclick={execute} disabled={isDisabled}>Add</Button>
      </div>
    </div>
  {/snippet}
</Modal>
