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

  let assemblyPath = $state('')
  let assemblyArgs = $state('')
  let processName = $state('')
  let ppid = $state(0)
  let processArgs = $state('')
  let amsiBypass = $state(false)
  let etwBypass = $state(false)
  let inProcess = $state(false)
  let arch = $state('')
  let runtime = $state('')
  let appDomain = $state('')
  let className = $state('')
  let methodName = $state('')
  let saveToLoot = $state(false)
  let lootName = $state('')
  let save = $state(false)
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    assemblyPath = values['local path to assembly'] || values['assembly'] || ''
    assemblyArgs = values['arguments'] || ''
    processName = values['process'] || ''
    ppid = values['ppid'] || 0
    processArgs = values['process-arguments'] || ''
    amsiBypass = values['amsi-bypass'] ?? false
    etwBypass = values['etw-bypass'] ?? false
    inProcess = values['in-process'] || false
    arch = values['arch'] || ''
    runtime = values['runtime'] || ''
    appDomain = values['app-domain'] || ''
    className = values['class'] || ''
    methodName = values['method'] || ''
    saveToLoot = values['loot'] || false
    lootName = values['name'] || ''
    save = false
    timeout = values['timeout'] || ''
  }


  function maybe(f, v) {
    if (v !== '' && v !== null && v !== undefined && v !== 0 && v !== false) return f(v)
    return ''
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['execute-assembly']
    if (assemblyPath) parts.push(quote(assemblyPath))
    if (assemblyArgs) parts.push(quote(assemblyArgs))
    if (processName) parts.push('--process', quote(processName))
    parts.push(maybe((v) => `--ppid ${v}`, ppid))
    if (processArgs) parts.push('--process-arguments', quote(processArgs))
    if (amsiBypass) parts.push('--amsi-bypass')
    if (etwBypass) parts.push('--etw-bypass')
    if (inProcess) parts.push('--in-process')
    if (arch) parts.push('--arch', arch)
    if (runtime) parts.push('--runtime', runtime)
    if (appDomain) parts.push('--app-domain', quote(appDomain))
    if (className) parts.push('--class', quote(className))
    if (methodName) parts.push('--method', quote(methodName))
    if (saveToLoot) {
      parts.push('--loot')
      if (lootName) parts.push('--name', quote(lootName))
    }
    if (save) parts.push('--save')
    if (timeout) parts.push('--timeout', timeout)
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Execute .NET Assembly" size="2xl" {onclose}>
  
    <p class="text-fg-muted text-sm mb-4">Load and execute a .NET assembly in a remote process.</p>

    <div class="mb-4">
      <FilePickerField bind:value={assemblyPath} label=".NET Assembly" />
    </div>

    <div class="mb-3">
      <TextField
        bind:value={assemblyArgs}
        label="Assembly Arguments"
        placeholder="Arguments to pass to the assembly entry point"
      />
    </div>

    <CollapsibleGroup title="Host Process" open={true}>
      <TextField
        bind:value={processName}
        label="Process"
        placeholder="notepad.exe, rundll32.exe, etc."
        description="Host process to inject the assembly into"
      />
      <PidPickerField bind:value={ppid} bind:processName label="PPID" sessionID={firstSessionID} />
      <TextField
        bind:value={processArgs}
        label="Process Arguments"
        placeholder="Arguments for the host process"
        description="Command-line arguments for the spawned process"
      />
    </CollapsibleGroup>

    <CollapsibleGroup title="Evasion" open={true}>
      <CheckboxField bind:checked={amsiBypass} label="Patch AMSI" description="Evade AMSI-based malware detection" />
      <CheckboxField bind:checked={etwBypass} label="Patch ETW" description="Evade Event Tracing for Windows" />
      <CheckboxField bind:checked={inProcess} label="Run in current process (no fork)" description="Don't spawn a sacrificial process" />
    </CollapsibleGroup>

    <CollapsibleGroup title="Runtime (advanced)" open={false}>
      <SelectField
        bind:value={arch}
        label="Architecture"
        placeholder="Default"
        options={[
          { value: 'x86', label: 'x86 (32-bit)' },
          { value: 'x64', label: 'x64 (64-bit)' },
        ]}
        description="Target process architecture"
      />
      <TextField
        bind:value={runtime}
        label="Runtime Version"
        placeholder="v4.0.30319"
        description=".NET runtime version for the assembly"
      />
      <TextField bind:value={appDomain} label="App Domain" placeholder="AppDomain name" />
      <TextField bind:value={className} label="Class Name" placeholder="Namespace.Class" />
      <TextField bind:value={methodName} label="Method Name" placeholder="Main" />
      <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
    </CollapsibleGroup>

    <CollapsibleGroup title="Output" open={false}>
      <CheckboxField bind:checked={saveToLoot} label="Save output to loot store" />
      {#if saveToLoot}
        <TextField
          bind:value={lootName}
          label="Loot Entry Name"
          placeholder="Default: assembly name"
          description="Name for the loot store entry"
        />
      {/if}
    </CollapsibleGroup>

    <div class="mb-4">
      <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
      <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview}</code>
    </div>
  
  {#snippet footer()}
    <div class="flex justify-between items-center">
    <PresetPicker
      commandPath="execute-assembly"
      currentValues={{
        'local path to assembly': assemblyPath,
        'arguments': assemblyArgs,
        'process': processName,
        'ppid': ppid,
        'process-arguments': processArgs,
        'amsi-bypass': amsiBypass,
        'etw-bypass': etwBypass,
        'in-process': inProcess,
        'arch': arch,
        'runtime': runtime,
        'app-domain': appDomain,
        'class': className,
        'method': methodName,
        'loot': saveToLoot,
        'name': lootName,
      }}
      onapply={(values) => {
        if (values['local path to assembly'] != null) assemblyPath = values['local path to assembly']
        if (values['arguments'] != null) assemblyArgs = values['arguments']
        if (values['process'] != null) processName = values['process']
        if (values['ppid'] != null) ppid = values['ppid']
        if (values['process-arguments'] != null) processArgs = values['process-arguments']
        if (values['amsi-bypass'] != null) amsiBypass = values['amsi-bypass']
        if (values['etw-bypass'] != null) etwBypass = values['etw-bypass']
        if (values['in-process'] != null) inProcess = values['in-process']
        if (values['arch'] != null) arch = values['arch']
        if (values['runtime'] != null) runtime = values['runtime']
        if (values['app-domain'] != null) appDomain = values['app-domain']
        if (values['class'] != null) className = values['class']
        if (values['method'] != null) methodName = values['method']
        if (values['loot'] != null) saveToLoot = values['loot']
        if (values['name'] != null) lootName = values['name']
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!assemblyPath}>Execute</Button>
    </div>
  </div>
  {/snippet}
</Modal>
