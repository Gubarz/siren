<script>
  import Modal from '$components/patterns/Modal.svelte'
  import TextField from '$components/forms/TextField.svelte'
  import { quote } from '$utils/shell.js'
  import CommandPreview from './CommandPreview.svelte'
  import CommandModalFooter from './CommandModalFooter.svelte'

  let {
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let username = $state('')
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    username = values['username'] || ''
    timeout = values['timeout'] || ''
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['impersonate']
    if (timeout) parts.push('--timeout', String(timeout))
    if (username) parts.push(quote(username))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Impersonate User" size="xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    Steal the primary token of an existing logon session for the given user. The current implant will operate as that user until you <code>rev2self</code>.
    Requires <code>SeDebugPrivilege</code> or SYSTEM.
  </p>

  <div class="mb-3">
    <TextField
      bind:value={username}
      label="Username"
      placeholder="DOMAIN\Administrator, hostname\admin, root, etc."
      description="An account that already has an interactive/service logon session on this host"
    />
  </div>

  <div class="mb-3">
    <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
  </div>

  <CommandPreview cmd={cmdPreview} />

  {#snippet footer()}
    <CommandModalFooter
      commandPath="impersonate"
      currentValues={{ username }}
      onapply={(values) => {
        if (values['username'] != null) username = values['username']
      }}
      primaryLabel="Impersonate"
      onprimary={execute}
      primaryDisabled={!username}
      oncancel={() => open = false}
    />
  {/snippet}
</Modal>
