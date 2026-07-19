<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import Button from '../../../../components/ui/Button.svelte'
  import HostPortField from '../../../../components/forms/HostPortField.svelte'
  import PresetPicker from '../../../../components/forms/PresetPicker.svelte'

  let {
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let bind = $state('127.0.0.1:8080')
  let remote = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    bind = values['bind'] || '127.0.0.1:8080'
    remote = values['remote'] || ''
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['portfwd', 'add']
    if (bind) parts.push('--bind', quote(bind))
    if (remote) parts.push('--remote', quote(remote))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Add Port Forward" size="xl" {onclose}>
  
    <p class="text-fg-muted text-sm mb-4">
      Forward traffic <strong>from your operator machine → through the implant → to a target service</strong>. Reachable services on the implant's network segment become reachable from your operator machine's <code>localhost</code>.
    </p>

    <p class="text-fg-muted text-xs mb-4 opacity-70">
      Example: <code>127.0.0.1:8080</code> ⇒ <code>10.0.0.5:80</code> — pointing your browser at <code>http://127.0.0.1:8080</code> hits the internal web server at <code>10.0.0.5:80</code> from the implant's perspective.
    </p>

    <div class="mb-3">
      <HostPortField
        bind:value={bind}
        label="Local bind (on operator)"
        hostPlaceholder="127.0.0.1"
        portPlaceholder="8080"
        description="Where you connect from"
      />
    </div>

    <div class="mb-3">
      <HostPortField
        bind:value={remote}
        label="Remote target (through implant)"
        hostPlaceholder="10.0.0.5"
        portPlaceholder="80"
        description="Where the traffic actually goes, as seen from the implant"
        required
      />
    </div>

    <div class="mb-4">
      <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
      <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview}</code>
    </div>
  
  {#snippet footer()}
    <div class="flex justify-between items-center">
    <PresetPicker
      commandPath="portfwd add"
      currentValues={{ 'bind': bind, 'remote': remote }}
      onapply={(values) => {
        if (values['bind'] != null) bind = values['bind']
        if (values['remote'] != null) remote = values['remote']
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!remote}>Add</Button>
    </div>
  </div>
  {/snippet}
</Modal>
