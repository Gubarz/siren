<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import Button from '../../../../components/ui/Button.svelte'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
  import SelectField from '../../../../components/forms/SelectField.svelte'
  import PresetPicker from '../../../../components/forms/PresetPicker.svelte'
  import PidPickerField from '../pickers/PidPickerField.svelte'

  let {
    firstSessionID = '',
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let pid = $state(0)
  let processName = $state('')
  let arch = $state('')
  let configName = $state('')
  let disableSgn = $state(false)
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    pid = values['pid'] || 0
    processName = values['process-name'] || ''
    arch = values['arch'] || ''
    configName = values['config'] || ''
    disableSgn = values['disable-sgn'] || false
    timeout = values['timeout'] || ''
  }


  let cmdPreview = $derived.by(() => {
    const parts = ['migrate']
    if (pid) parts.push('--pid', String(pid))
    if (processName) parts.push('--process-name', quote(processName))
    if (arch) parts.push('--arch', arch)
    if (configName) parts.push('--config', quote(configName))
    if (disableSgn) parts.push('--disable-sgn')
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
        bind:value={arch}
        label="Architecture"
        placeholder="Auto-detect"
        options={[
          { value: 'amd64', label: 'amd64 (64-bit)' },
          { value: '386', label: '386 (32-bit)' },
        ]}
        description="Match the target process. Leave blank to auto-detect."
      />
    </div>

    <div class="mb-3">
      <TextField
        bind:value={configName}
        label="Implant profile"
        placeholder="Name of an existing implant profile"
        description="Which implant configuration to migrate as. Blank re-uses the current implant."
      />
    </div>

    <CollapsibleGroup title="Advanced" open={false}>
      <CheckboxField bind:checked={disableSgn} label="Disable SGN encoding" description="Skip Shikata Ga Nai encoding of the migration shellcode" />
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
      currentValues={{ 'pid': pid, 'process-name': processName, 'arch': arch, 'config': configName, 'disable-sgn': disableSgn }}
      onapply={(values) => {
        if (values['pid'] != null) pid = values['pid']
        if (values['process-name'] != null) processName = values['process-name']
        if (values['arch'] != null) arch = values['arch']
        if (values['config'] != null) configName = values['config']
        if (values['disable-sgn'] != null) disableSgn = values['disable-sgn']
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!pid && !processName}>Migrate</Button>
    </div>
  </div>
  {/snippet}
</Modal>
