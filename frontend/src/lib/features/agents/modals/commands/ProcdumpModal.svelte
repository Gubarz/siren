<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
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

  let pid = $state(0)
  let processName = $state('')
  let outputFile = $state('')
  let saveToLoot = $state(true)
  let lootName = $state('')
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    pid = values['pid'] || 0
    processName = values['name'] || ''
    outputFile = values['save'] || ''
    saveToLoot = values['loot'] ?? true
    lootName = values['loot-name'] || ''
    timeout = values['timeout'] || ''
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['procdump']
    if (pid) parts.push('--pid', String(pid))
    if (processName) parts.push('--name', quote(processName))
    if (outputFile) parts.push('--save', quote(outputFile))
    if (saveToLoot) {
      parts.push('--loot')
      if (lootName) parts.push('--loot-name', quote(lootName))
    }
    if (timeout) parts.push('--timeout', String(timeout))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Process Memory Dump" size="2xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">Dump the memory of a target process. Great for LSASS credential extraction (open the dump in mimikatz / pypykatz offline).</p>

  <CollapsibleGroup title="Target process" open={true}>
    <PidPickerField bind:value={pid} bind:processName label="Process PID" sessionID={firstSessionID} />
    <TextField
      bind:value={processName}
      label="Process name"
      placeholder="lsass.exe, chrome.exe, etc."
      description="Alternative to PID — dumps the first process matching this name"
    />
  </CollapsibleGroup>

  <CollapsibleGroup title="Where to save" open={true}>
    <CheckboxField bind:checked={saveToLoot} label="Save to loot store" description="Sends the dump to the teamserver loot store so the whole team can grab it" />
    {#if saveToLoot}
      <TextField bind:value={lootName} label="Loot entry name" placeholder="Default: <process>.dmp" />
    {/if}
    <TextField
      bind:value={outputFile}
      label="Also save to disk (operator)"
      placeholder="Local path for a copy"
      description="Blank = don't write to disk (loot only)"
    />
  </CollapsibleGroup>

  <CollapsibleGroup title="Advanced" open={false}>
    <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
  </CollapsibleGroup>

  <CommandPreview cmd={cmdPreview} />

  {#snippet footer()}
    <CommandModalFooter
      commandPath="procdump"
      currentValues={{ 'pid': pid, 'name': processName, 'save': outputFile, 'loot': saveToLoot, 'loot-name': lootName }}
      onapply={(values) => {
        if (values['pid'] != null) pid = values['pid']
        if (values['name'] != null) processName = values['name']
        if (values['save'] != null) outputFile = values['save']
        if (values['loot'] != null) saveToLoot = values['loot']
        if (values['loot-name'] != null) lootName = values['loot-name']
      }}
      primaryLabel="Dump"
      onprimary={execute}
      primaryDisabled={!pid && !processName}
      oncancel={() => open = false}
    />
  {/snippet}
</Modal>
