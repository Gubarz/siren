<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import Button from '../../../../components/ui/Button.svelte'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
  import SelectField from '../../../../components/forms/SelectField.svelte'
  import FilePickerField from '../../../../components/forms/FilePickerField.svelte'
  import PresetPicker from '../../../../components/forms/PresetPicker.svelte'
  import PidPickerField from '../pickers/PidPickerField.svelte'

  let {
    firstSessionID = '',
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let scPath = $state('')
  let injectionMode = $state('self')
  let pid = $state(0)
  let processName = $state('')
  let architecture = $state('amd64')
  let rwxPages = $state(false)
  let interactive = $state(false)
  let shikataGaNai = $state(false)
  let iterations = $state(1)
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    scPath = values['local path to shellcode'] || values['shellcode'] || ''
    injectionMode = values['pid'] ? 'pid' : (values['process'] ? 'process' : 'self')
    pid = values['pid'] || 0
    processName = values['process'] || ''
    architecture = values['architecture'] || 'amd64'
    rwxPages = values['rwx-pages'] || false
    interactive = values['interactive'] || false
    shikataGaNai = values['shikata-ga-nai'] || false
    iterations = values['iterations'] || 1
    timeout = values['timeout'] || ''
  }


  let cmdPreview = $derived.by(() => {
    const parts = ['execute-shellcode']
    if (architecture) parts.push('--architecture', architecture)
    if (injectionMode === 'pid' && pid) parts.push('--pid', String(pid))
    else if (injectionMode === 'process' && processName) parts.push('--process', quote(processName))
    if (rwxPages) parts.push('--rwx-pages')
    if (interactive) parts.push('--interactive')
    if (shikataGaNai) {
      parts.push('--shikata-ga-nai')
      if (iterations && iterations !== 1) parts.push('--iterations', String(iterations))
    }
    if (timeout) parts.push('--timeout', String(timeout))
    if (scPath) parts.push(quote(scPath))
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Execute Shellcode" size="2xl" {onclose}>
  
    <p class="text-fg-muted text-sm mb-4">Inject raw shellcode into a target process (or the current implant).</p>

    <div class="mb-4">
      <FilePickerField bind:value={scPath} label="Shellcode file" />
    </div>

    <CollapsibleGroup title="Injection target" open={true}>
      <SelectField
        bind:value={injectionMode}
        label="Where to inject"
        options={[
          { value: 'self', label: 'Current implant process (in-process)' },
          { value: 'pid', label: 'Existing PID' },
          { value: 'process', label: 'Spawn new process by name' },
        ]}
      />
      {#if injectionMode === 'pid'}
        <PidPickerField bind:value={pid} label="Target PID" sessionID={firstSessionID} />
      {:else if injectionMode === 'process'}
        <TextField
          bind:value={processName}
          label="Process to spawn"
          placeholder="notepad.exe, rundll32.exe, etc."
          description="Sacrificial process the shellcode is injected into"
        />
      {/if}
      <SelectField
        bind:value={architecture}
        label="Architecture"
        options={[
          { value: 'amd64', label: 'amd64 (64-bit)' },
          { value: '386', label: '386 (32-bit)' },
        ]}
      />
    </CollapsibleGroup>

    <CollapsibleGroup title="Advanced" open={false}>
      <CheckboxField bind:checked={rwxPages} label="Use RWX pages" description="Allocate as read-write-execute (noisier, more compatible)" />
      <CheckboxField bind:checked={interactive} label="Interactive shellcode" description="Wire up stdio (for staged shells)" />
      <CheckboxField bind:checked={shikataGaNai} label="Encode with Shikata Ga Nai" description="msf-style polymorphic encoder" />
      {#if shikataGaNai}
        <TextField bind:value={iterations} label="Encoder iterations" type="number" />
      {/if}
      <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
    </CollapsibleGroup>

    <div class="mb-4">
      <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
      <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview}</code>
    </div>
  
  {#snippet footer()}
    <div class="flex justify-between items-center">
    <PresetPicker
      commandPath="execute-shellcode"
      currentValues={{
        'local path to shellcode': scPath,
        'pid': injectionMode === 'pid' ? pid : 0,
        'process': injectionMode === 'process' ? processName : '',
        'architecture': architecture,
        'rwx-pages': rwxPages,
        'interactive': interactive,
        'shikata-ga-nai': shikataGaNai,
        'iterations': iterations,
      }}
      onapply={(values) => {
        if (values['local path to shellcode'] != null) scPath = values['local path to shellcode']
        if (values['pid']) { pid = values['pid']; injectionMode = 'pid' }
        else if (values['process']) { processName = values['process']; injectionMode = 'process' }
        if (values['architecture'] != null) architecture = values['architecture']
        if (values['rwx-pages'] != null) rwxPages = values['rwx-pages']
        if (values['interactive'] != null) interactive = values['interactive']
        if (values['shikata-ga-nai'] != null) shikataGaNai = values['shikata-ga-nai']
        if (values['iterations'] != null) iterations = values['iterations']
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!scPath}>Execute</Button>
    </div>
  </div>
  {/snippet}
</Modal>
