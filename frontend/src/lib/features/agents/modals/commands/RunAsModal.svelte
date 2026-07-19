<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import Button from '../../../../components/ui/Button.svelte'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
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
  let program = $state('')
  let programArgs = $state('')
  let netonly = $state(true)
  let showWindow = $state(false)
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    username = values['username'] || ''
    password = values['password'] || ''
    domain = values['domain'] || ''
    program = values['program'] || ''
    programArgs = values['args'] || ''
    netonly = values['net-only'] ?? true
    showWindow = values['show-window'] || false
    timeout = values['timeout'] || ''
  }


  let cmdPreview = $derived.by(() => {
    const parts = ['runas']
    if (username) parts.push('--username', quote(username))
    if (password) parts.push('--password', quote(password))
    if (domain) parts.push('--domain', quote(domain))
    if (program) parts.push('--program', quote(program))
    if (programArgs) parts.push('--args', quote(programArgs))
    if (netonly) parts.push('--net-only')
    if (showWindow) parts.push('--show-window')
    if (timeout) parts.push('--timeout', String(timeout))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Run As User" size="2xl" {onclose}>
  
    <p class="text-fg-muted text-sm mb-4">
      Launch a program under a different user's credentials. Similar to Windows <code>runas.exe</code>. The spawned process runs with its own token — the parent implant identity is unchanged.
    </p>

    <CollapsibleGroup title="Credentials" open={true}>
      <TextField bind:value={username} label="Username" placeholder="Administrator" />
      <TextField bind:value={password} label="Password" type="password" placeholder="Password" />
      <TextField
        bind:value={domain}
        label="Domain"
        placeholder="CORP or . for local"
        description="Blank/dot = local account, otherwise a domain name"
      />
      <CheckboxField
        bind:checked={netonly}
        label="Net-only (don't validate credentials locally)"
        description="Uses these creds only for network resource access. Quieter — no local logon event."
      />
    </CollapsibleGroup>

    <CollapsibleGroup title="Program to launch" open={true}>
      <TextField
        bind:value={program}
        label="Program"
        placeholder="cmd.exe, powershell.exe, C:\\Windows\\System32\\net.exe"
      />
      <TextField
        bind:value={programArgs}
        label="Arguments"
        placeholder="/c whoami /all"
      />
      <CheckboxField
        bind:checked={showWindow}
        label="Show window"
        description="Otherwise the child process is spawned hidden (default)"
      />
    </CollapsibleGroup>

    <CollapsibleGroup title="Advanced" open={false}>
      <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
    </CollapsibleGroup>

    <div class="mb-4">
      <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
      <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview.replace(/--password "[^"]*"/, '--password "***"')}</code>
    </div>
  
  {#snippet footer()}
    <div class="flex justify-between items-center">
    <PresetPicker
      commandPath="runas"
      currentValues={{
        'username': username,
        'domain': domain,
        'program': program,
        'args': programArgs,
        'net-only': netonly,
        'show-window': showWindow,
      }}
      onapply={(values) => {
        if (values['username'] != null) username = values['username']
        if (values['domain'] != null) domain = values['domain']
        if (values['program'] != null) program = values['program']
        if (values['args'] != null) programArgs = values['args']
        if (values['net-only'] != null) netonly = values['net-only']
        if (values['show-window'] != null) showWindow = values['show-window']
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!username || !password || !program}>Run</Button>
    </div>
  </div>
  {/snippet}
</Modal>
