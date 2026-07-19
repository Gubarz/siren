<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import Button from '../../../../components/ui/Button.svelte'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
  import FilePickerField from '../../../../components/forms/FilePickerField.svelte'
  import PresetPicker from '../../../../components/forms/PresetPicker.svelte'

  let {
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let dllPath = $state('')
  let entryPoint = $state('')
  let processName = $state('')
  let args = $state('')
  let isUnmanaged = $state(false)
  let runtime = $state('')
  let save = $state(false)

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    dllPath = values['local path to dll'] || values['dll'] || ''
    entryPoint = values['entry-point'] || ''
    processName = values['process-name'] || ''
    args = values['args'] || ''
    isUnmanaged = values['is-unmanaged'] || false
    runtime = values['runtime'] || ''
    save = false
  }


  let cmdPreview = $derived.by(() => {
    const parts = ['sideload']
    if (dllPath) parts.push(quote(dllPath))
    if (entryPoint) parts.push('--entry-point', quote(entryPoint))
    if (args) parts.push('--args', quote(args))
    if (processName) parts.push('--process-name', quote(processName))
    if (isUnmanaged) parts.push('--is-unmanaged')
    if (runtime) parts.push('--runtime', runtime)
    if (save) parts.push('--save')
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Sideload DLL" size="2xl" {onclose}>
  
    <p class="text-fg-muted text-sm mb-4">Load a shared library (DLL) into a remote process.</p>

    <div class="mb-4">
      <FilePickerField bind:value={dllPath} label="DLL to sideload" />
    </div>

    <div class="mb-3">
      <TextField
        bind:value={entryPoint}
        label="Entry Point"
        placeholder="Name of the exported function to call"
      />
    </div>

    <CollapsibleGroup title="Host Process" open={true}>
      <TextField
        bind:value={processName}
        label="Process"
        placeholder="notepad.exe, rundll32.exe, etc."
        description="Host process to inject the DLL into"
      />
      <TextField
        bind:value={args}
        label="Arguments"
        placeholder="Arguments for the DLL entry point"
      />
      <CheckboxField
        bind:checked={isUnmanaged}
        label="Use unmanaged process (no fork+run)"
        description="Load the DLL directly without forking a managed process"
      />
    </CollapsibleGroup>

    <CollapsibleGroup title="Runtime (advanced)" open={false}>
      <TextField
        bind:value={runtime}
        label="Runtime"
        placeholder="v4.0.30319"
        description=".NET runtime version"
      />
    </CollapsibleGroup>

    <CollapsibleGroup title="Output" open={false}>
      <CheckboxField bind:checked={save} label="Save to disk on target" />
    </CollapsibleGroup>

    <div class="mb-4">
      <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
      <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview}</code>
    </div>
  
  {#snippet footer()}
    <div class="flex justify-between items-center">
    <PresetPicker
      commandPath="sideload"
      currentValues={{
        'local path to dll': dllPath,
        'entry-point': entryPoint,
        'process-name': processName,
        'args': args,
        'is-unmanaged': isUnmanaged,
        'runtime': runtime,
      }}
      onapply={(values) => {
        if (values['local path to dll'] != null) dllPath = values['local path to dll']
        if (values['entry-point'] != null) entryPoint = values['entry-point']
        if (values['process-name'] != null) processName = values['process-name']
        if (values['args'] != null) args = values['args']
        if (values['is-unmanaged'] != null) isUnmanaged = values['is-unmanaged']
        if (values['runtime'] != null) runtime = values['runtime']
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!dllPath}>Execute</Button>
    </div>
  </div>
  {/snippet}
</Modal>
