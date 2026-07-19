<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import Button from '../../../../components/ui/Button.svelte'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import SelectField from '../../../../components/forms/SelectField.svelte'
  import PresetPicker from '../../../../components/forms/PresetPicker.svelte'

  let {
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let username = $state('')
  let password = $state('')
  let domain = $state('')
  let logonType = $state('LOGON_NEW_CREDENTIALS')
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    username = values['username'] || ''
    password = values['password'] || ''
    domain = values['domain'] || ''
    logonType = values['logon-type'] || 'LOGON_NEW_CREDENTIALS'
    timeout = values['timeout'] || ''
  }


  let cmdPreview = $derived.by(() => {
    const parts = ['make-token']
    if (username) parts.push('--username', quote(username))
    if (password) parts.push('--password', quote(password))
    if (domain) parts.push('--domain', quote(domain))
    if (logonType) parts.push('--logon-type', logonType)
    if (timeout) parts.push('--timeout', String(timeout))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Make Token" size="xl" {onclose}>
  
    <p class="text-fg-muted text-sm mb-4">
      Create a new access token from clear-text credentials. Typical use: <em>net-only</em> logon so subsequent network commands authenticate as the given user (Kerberos / SMB / WinRM) without changing local identity.
      Credentials are validated locally.
    </p>

    <div class="mb-3">
      <TextField
        bind:value={username}
        label="Username"
        placeholder="Administrator"
      />
    </div>

    <div class="mb-3">
      <TextField
        bind:value={password}
        label="Password"
        type="password"
        placeholder="Password"
      />
    </div>

    <div class="mb-3">
      <TextField
        bind:value={domain}
        label="Domain"
        placeholder="CORP or . for local machine"
        description="Blank/dot = local account. Set for domain accounts."
      />
    </div>

    <CollapsibleGroup title="Advanced" open={false}>
      <SelectField
        bind:value={logonType}
        label="Logon type"
        options={[
          { value: 'LOGON_NEW_CREDENTIALS', label: 'NEW_CREDENTIALS (net-only, quietest)' },
          { value: 'LOGON_INTERACTIVE', label: 'INTERACTIVE' },
          { value: 'LOGON_NETWORK', label: 'NETWORK' },
          { value: 'LOGON_BATCH', label: 'BATCH' },
          { value: 'LOGON_SERVICE', label: 'SERVICE' },
          { value: 'LOGON_NETWORK_CLEARTEXT', label: 'NETWORK_CLEARTEXT' },
        ]}
        description="NEW_CREDENTIALS (runas /netonly) is the default — validates only on remote access, doesn't log a local logon event."
      />
      <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
    </CollapsibleGroup>

    <div class="mb-4">
      <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
      <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview.replace(/--password "[^"]*"/, '--password "***"')}</code>
    </div>
  
  {#snippet footer()}
    <div class="flex justify-between items-center">
    <PresetPicker
      commandPath="make-token"
      currentValues={{ 'username': username, 'domain': domain, 'logon-type': logonType }}
      onapply={(values) => {
        if (values['username'] != null) username = values['username']
        if (values['domain'] != null) domain = values['domain']
        if (values['logon-type'] != null) logonType = values['logon-type']
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!username || !password}>Make Token</Button>
    </div>
  </div>
  {/snippet}
</Modal>
