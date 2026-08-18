<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import RemoteFilePickerField from '../pickers/RemoteFilePickerField.svelte'
  import CommandPreview from './CommandPreview.svelte'
  import CommandModalFooter from './CommandModalFooter.svelte'

  let {
    sessionID = '',
    firstSessionID = '',
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let remoteFilePath = $state('')
  let profile = $state('')
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    remoteFilePath = values['remote file'] || values['file'] || ''
    profile = values['profile'] || ''
    timeout = values['timeout'] || ''
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['backdoor']
    if (profile) parts.push('--profile', quote(profile))
    if (timeout) parts.push('--timeout', String(timeout))
    if (remoteFilePath) parts.push(quote(remoteFilePath))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Backdoor Remote Binary" size="2xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    Infect an existing binary on the target with implant shellcode. Sliver downloads the file, patches in the shellcode from the chosen profile, and uploads it back to the same path.
    Persistence idea: pick a legitimate service binary or a scheduled-task target.
  </p>

  <div class="mb-3">
    <RemoteFilePickerField
      bind:value={remoteFilePath}
      sessionID={firstSessionID || sessionID}
      label="Remote file to backdoor"
      placeholder="C:\\Windows\\System32\\some.exe"
      description="Path on the target — must already exist and be writable by the implant"
      mode="file"
    />
  </div>

  <div class="mb-3">
    <TextField
      bind:value={profile}
      label="Implant profile"
      placeholder="Name of an existing implant profile"
      description="Which implant config to embed as shellcode. Blank uses the default profile."
    />
  </div>

  <CollapsibleGroup title="Advanced" open={false}>
    <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
  </CollapsibleGroup>

  <CommandPreview cmd={cmdPreview} />

  {#snippet footer()}
    <CommandModalFooter
      commandPath="backdoor"
      currentValues={{ 'remote file': remoteFilePath, 'profile': profile }}
      onapply={(values) => {
        if (values['remote file'] != null) remoteFilePath = values['remote file']
        if (values['profile'] != null) profile = values['profile']
      }}
      primaryLabel="Backdoor"
      onprimary={execute}
      primaryDisabled={!remoteFilePath}
      oncancel={() => open = false}
    />
  {/snippet}
</Modal>
