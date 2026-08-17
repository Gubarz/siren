<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import Button from '../../../../components/ui/Button.svelte'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import SelectField from '../../../../components/forms/SelectField.svelte'
  import PresetPicker from '../../../../components/forms/PresetPicker.svelte'
  import PidPickerField from '../pickers/PidPickerField.svelte'
  import { useResource } from '../../../../stores/lib/createResource.svelte.js'
  import { shellcodeEncoders } from '../../../../stores/resources/shellcodeEncoders.svelte.js'

  let {
    firstSessionID = '',
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  useResource(shellcodeEncoders)

  let pid = $state(0)
  let processName = $state('')
  let encoder = $state('none')
  let timeout = $state('')

  // The remote console can only answer the encoder question interactively
  // (it renders a forms.Select prompt in xterm), so the modal must always
  // pass --shellcode-encoder explicitly. "none" matches the default choice
  // of sliver's own interactive prompt.
  let encoderOptions = $derived([
    { value: 'none', label: 'None' },
    ...[...new Set((shellcodeEncoders.data || []).map((item) => item.name))]
      .sort()
      .map((name) => ({ value: name, label: name })),
  ])

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    pid = values['pid'] || 0
    processName = values['process-name'] || ''
    encoder = values['shellcode-encoder'] || 'none'
    timeout = values['timeout'] || ''
  }


  let cmdPreview = $derived.by(() => {
    const parts = ['migrate']
    if (pid) parts.push('--pid', String(pid))
    if (processName) parts.push('--process-name', quote(processName))
    parts.push('--shellcode-encoder', encoder)
    if (timeout) parts.push('--timeout', String(timeout))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Migrate to Process" size="2xl" {onclose}>
  
    <p class="text-fg-muted text-sm mb-4">Move the current implant into another running process. The current process is left behind and a new session opens under the target PID.</p>

    <div class="mb-3">
      <PidPickerField bind:value={pid} label="Target PID (-p)" sessionID={firstSessionID} />
    </div>

    <div class="mb-3">
      <TextField
        bind:value={processName}
        label="Or process name (-n)"
        placeholder="e.g. explorer.exe"
        description="Match by executable name instead of PID. Ignored if PID is set."
      />
    </div>

    <div class="mb-3">
      <SelectField
        bind:value={encoder}
        label="Shellcode encoder"
        options={encoderOptions}
        description="Encoding applied to the migration shellcode. None matches the console default."
      />
    </div>

    <CollapsibleGroup title="Advanced" open={false}>
      <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
    </CollapsibleGroup>

    <div class="mb-4">
      <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
      <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview}</code>
    </div>
  
  {#snippet footer()}
    <div class="flex justify-between items-center">
    <PresetPicker
      commandPath="migrate"
      currentValues={{ 'pid': pid, 'process-name': processName, 'shellcode-encoder': encoder, 'timeout': timeout }}
      onapply={(values) => {
        if (values['pid'] != null) pid = values['pid']
        if (values['process-name'] != null) processName = values['process-name']
        if (values['shellcode-encoder'] != null) encoder = values['shellcode-encoder'] || 'none'
        if (values['timeout'] != null) timeout = values['timeout']
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!pid && !processName}>Migrate</Button>
    </div>
  </div>
  {/snippet}
</Modal>
