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

  let bind = $state('0.0.0.0:4444')
  let remote = $state('127.0.0.1:4444')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    bind = values['bind'] || '0.0.0.0:4444'
    remote = values['remote'] || '127.0.0.1:4444'
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['rportfwd', 'add']
    if (bind) parts.push('--bind', quote(bind))
    if (remote) parts.push('--remote', quote(remote))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Add Reverse Port Forward" size="xl" {onclose}>
  
    <p class="text-fg-muted text-sm mb-4">
      Open a listener <strong>on the implant</strong> that forwards connections back <strong>to the operator/teamserver</strong>. Use this to expose an operator-side service (Responder, Impacket <code>ntlmrelayx</code>, a fake update server) to the target's network.
    </p>

    <p class="text-fg-muted text-xs mb-4 opacity-70">
      Example: implant binds <code>0.0.0.0:4444</code>, forwards to <code>127.0.0.1:8080</code> on the operator. Anyone on the target LAN that hits the implant on port 4444 lands on your local <code>:8080</code>.
    </p>

    <div class="mb-3">
      <HostPortField
        bind:value={bind}
        label="Implant listen (on target machine)"
        hostPlaceholder="0.0.0.0"
        portPlaceholder="4444"
        description="Interface + port the implant listens on. 0.0.0.0 = all interfaces."
        required
      />
    </div>

    <div class="mb-3">
      <HostPortField
        bind:value={remote}
        label="Forward to (on operator/server)"
        hostPlaceholder="127.0.0.1"
        portPlaceholder="4444"
        description="Where connections are relayed to, from the server's perspective"
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
      commandPath="rportfwd add"
      currentValues={{ 'bind': bind, 'remote': remote }}
      onapply={(values) => {
        if (values['bind'] != null) bind = values['bind']
        if (values['remote'] != null) remote = values['remote']
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!bind || !remote}>Add</Button>
    </div>
  </div>
  {/snippet}
</Modal>
