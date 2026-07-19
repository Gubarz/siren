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

  let hostProcess = $state('spoolsv.exe')
  let profileName = $state('')
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    hostProcess = values['process'] || 'spoolsv.exe'
    profileName = values['config'] || ''
    timeout = values['timeout'] || ''
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['getsystem']
    if (hostProcess) parts.push('--process', quote(hostProcess))
    if (profileName) parts.push('--config', quote(profileName))
    if (timeout) parts.push('--timeout', String(timeout))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Get SYSTEM" size="xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    Elevate the current session to <code>NT AUTHORITY\SYSTEM</code> by spawning a new session under a chosen host process, using named-pipe impersonation.
    Requires the current implant to already hold <code>SeDebugPrivilege</code> or be running as a high-integrity admin.
  </p>

  <div class="mb-3">
    <TextField
      bind:value={hostProcess}
      label="Host process"
      placeholder="spoolsv.exe"
      description="A process running as SYSTEM whose token we impersonate. Common picks: spoolsv.exe, lsass.exe, winlogon.exe."
    />
  </div>

  <div class="mb-3">
    <TextField
      bind:value={profileName}
      label="Implant profile"
      placeholder="Blank = use current implant config"
      description="Which implant profile the spawned SYSTEM session should use"
    />
  </div>

  <div class="mb-3">
    <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
  </div>

  <CommandPreview cmd={cmdPreview} />

  {#snippet footer()}
    <CommandModalFooter
      commandPath="getsystem"
      currentValues={{ process: hostProcess, config: profileName }}
      onapply={(values) => {
        if (values['process'] != null) hostProcess = values['process']
        if (values['config'] != null) profileName = values['config']
      }}
      primaryLabel="Elevate"
      onprimary={execute}
      primaryDisabled={!hostProcess}
      oncancel={() => open = false}
    />
  {/snippet}
</Modal>
