<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
  import FilePickerField from '../../../../components/forms/FilePickerField.svelte'
  import PidPickerField from '../pickers/PidPickerField.svelte'
  import CommandPreview from './CommandPreview.svelte'
  import CommandModalFooter from './CommandModalFooter.svelte'

  let {
    firstSessionID = '',
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let dllPath = $state('')
  let processName = $state('notepad.exe')
  let args = $state('')
  let ppid = $state(0)
  let killSpawning = $state(false)
  let offset = $state('')
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    dllPath = values['local path to dll'] || values['dll'] || ''
    processName = values['process-name'] || 'notepad.exe'
    args = values['args'] || ''
    ppid = values['ppid'] || 0
    killSpawning = values['kill'] || false
    offset = values['offset'] || ''
    timeout = values['timeout'] || ''
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['spawndll']
    if (processName) parts.push('--process-name', quote(processName))
    if (args) parts.push('--args', quote(args))
    if (ppid) parts.push('--ppid', String(ppid))
    if (killSpawning) parts.push('--kill')
    if (offset) parts.push('--offset', String(offset))
    if (timeout) parts.push('--timeout', String(timeout))
    if (dllPath) parts.push(quote(dllPath))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Spawn DLL" size="2xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">Spawn a reflective DLL in a new sacrificial process. Unlike <code>sideload</code>, this doesn't fork+run — the DLL is loaded directly into the spawned process.</p>

  <div class="mb-4">
    <FilePickerField bind:value={dllPath} label="Reflective DLL" />
  </div>

  <CollapsibleGroup title="Host Process" open={true}>
    <TextField
      bind:value={processName}
      label="Process to spawn"
      placeholder="notepad.exe, rundll32.exe, etc."
      description="A sacrificial process the DLL runs in"
    />
    <PidPickerField bind:value={ppid} label="Parent PID (spoof)" sessionID={firstSessionID} />
    <TextField
      bind:value={args}
      label="Arguments"
      placeholder="Arguments passed to the DLL entry point"
    />
    <CheckboxField
      bind:checked={killSpawning}
      label="Kill spawning process after execution"
      description="Terminates the host process once the DLL returns"
    />
  </CollapsibleGroup>

  <CollapsibleGroup title="Advanced" open={false}>
    <TextField
      bind:value={offset}
      label="DLL offset"
      type="number"
      description="Entry point offset within the DLL (default 0)"
    />
    <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
  </CollapsibleGroup>

  <CommandPreview cmd={cmdPreview} />

  {#snippet footer()}
    <CommandModalFooter
      commandPath="spawndll"
      currentValues={{
        'local path to dll': dllPath,
        'process-name': processName,
        'args': args,
        'ppid': ppid,
        'kill': killSpawning,
        'offset': offset,
      }}
      onapply={(values) => {
        if (values['local path to dll'] != null) dllPath = values['local path to dll']
        if (values['process-name'] != null) processName = values['process-name']
        if (values['args'] != null) args = values['args']
        if (values['ppid'] != null) ppid = values['ppid']
        if (values['kill'] != null) killSpawning = values['kill']
        if (values['offset'] != null) offset = values['offset']
      }}
      primaryLabel="Spawn"
      onprimary={execute}
      primaryDisabled={!dllPath}
      oncancel={() => open = false}
    />
  {/snippet}
</Modal>
