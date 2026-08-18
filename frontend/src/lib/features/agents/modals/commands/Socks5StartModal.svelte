<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
  import CommandPreview from './CommandPreview.svelte'
  import CommandModalFooter from './CommandModalFooter.svelte'

  let {
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let host = $state('127.0.0.1')
  let port = $state('1080')
  let useAuth = $state(false)
  let user = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    host = values['host'] || '127.0.0.1'
    port = values['port'] || '1080'
    useAuth = !!values['user']
    user = values['user'] || ''
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['socks5', 'start']
    if (host) parts.push('--host', host)
    if (port) parts.push('--port', String(port))
    if (useAuth && user) parts.push('--user', quote(user))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Start SOCKS5 Proxy" size="xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    Open a local SOCKS5 listener on your operator machine. Traffic gets tunneled through the implant, so tools like <code>proxychains</code>, browsers, and <code>nmap</code> can reach the target's internal network.
  </p>

  <div class="mb-3">
    <TextField
      bind:value={host}
      label="Bind host"
      placeholder="127.0.0.1"
      description="Interface the SOCKS listener binds to on your operator machine"
    />
  </div>

  <div class="mb-3">
    <TextField
      bind:value={port}
      label="Local port"
      type="number"
      placeholder="1080"
    />
  </div>

  <CollapsibleGroup title="Authentication (advanced)" open={false}>
    <CheckboxField
      bind:checked={useAuth}
      label="Require username/password auth"
      description="WARNING: credentials are tunneled to the implant and recoverable from its memory"
    />
    {#if useAuth}
      <TextField bind:value={user} label="Username" placeholder="operator" />
    {/if}
  </CollapsibleGroup>

  <CommandPreview cmd={cmdPreview} />

  {#snippet footer()}
    <CommandModalFooter
      commandPath="socks5 start"
      currentValues={{ 'host': host, 'port': port, 'user': useAuth ? user : '' }}
      onapply={(values) => {
        if (values['host'] != null) host = values['host']
        if (values['port'] != null) port = values['port']
        if (values['user']) { user = values['user']; useAuth = true }
      }}
      primaryLabel="Start"
      onprimary={execute}
      primaryDisabled={!port}
      oncancel={() => open = false}
    />
  {/snippet}
</Modal>
