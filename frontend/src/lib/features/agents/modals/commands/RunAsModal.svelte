<script>
  import { quote } from '../../../../utils/shell.js'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
  import CredentialPicker from '../CredentialPicker.svelte'
  import CommandModalFrame from './CommandModalFrame.svelte'
  import { createCredentialForm } from '../credentialForm.svelte.js'

  let { open = $bindable(false), onexecute, ...rest } = $props()

  const credentials = createCredentialForm()
  let program = $state('')
  let programArgs = $state('')
  let netonly = $state(true)
  let showWindow = $state(false)

  function resetForm(values) {
    credentials.reset(values)
    program = values['program'] || ''
    programArgs = values['args'] || ''
    netonly = values['net-only'] ?? true
    showWindow = values['show-window'] || false
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['runas']
    if (credentials.fields.username) parts.push('--username', quote(credentials.fields.username))
    if (credentials.fields.password) parts.push('--password', quote(credentials.fields.password))
    if (credentials.fields.domain) parts.push('--domain', quote(credentials.fields.domain))
    if (program) parts.push('--program', quote(program))
    if (programArgs) parts.push('--args', quote(programArgs))
    if (netonly) parts.push('--net-only')
    if (showWindow) parts.push('--show-window')
    if (credentials.fields.timeout) parts.push('--timeout', String(credentials.fields.timeout))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<CommandModalFrame
  bind:open
  title="Run As User"
  size="2xl"
  {cmdPreview}
  commandPath="runas"
  currentValues={{ username: credentials.fields.username, domain: credentials.fields.domain, program, 'args': programArgs, 'net-only': netonly, 'show-window': showWindow }}
  onapply={(values) => {
    if (values['username'] != null) credentials.fields.username = values['username']
    if (values['domain'] != null) credentials.fields.domain = values['domain']
    if (values['program'] != null) program = values['program']
    if (values['args'] != null) programArgs = values['args']
    if (values['net-only'] != null) netonly = values['net-only']
    if (values['show-window'] != null) showWindow = values['show-window']
  }}
  primaryLabel="Run"
  onprimary={execute}
  primaryDisabled={!credentials.fields.username || !credentials.fields.password || !program}
  onreset={resetForm}
  {...rest}
>
  <p class="text-fg-muted text-sm mb-4">
    Launch a program under a different user's credentials. Similar to Windows <code>runas.exe</code>. The spawned process runs with its own token — the parent implant identity is unchanged.
  </p>

  <CredentialPicker bind:username={credentials.fields.username} bind:password={credentials.fields.password} bind:domain={credentials.fields.domain} bind:timeout={credentials.fields.timeout} />

  <CollapsibleGroup title="Program to launch" open={true}>
    <TextField bind:value={program} label="Program" placeholder="cmd.exe, powershell.exe, C:\\Windows\\System32\\net.exe" />
    <TextField bind:value={programArgs} label="Arguments" placeholder="/c whoami /all" />
    <CheckboxField bind:checked={showWindow} label="Show window" description="Otherwise the child process is spawned hidden (default)" />
    <CheckboxField bind:checked={netonly} label="Net-only (don't validate credentials locally)" description="Uses these creds only for network resource access. Quieter — no local logon event." />
  </CollapsibleGroup>
</CommandModalFrame>
