<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import Button from '../../../../components/ui/Button.svelte'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import SelectField from '../../../../components/forms/SelectField.svelte'
  import FilePickerField from '../../../../components/forms/FilePickerField.svelte'
  import PresetPicker from '../../../../components/forms/PresetPicker.svelte'
  import RemoteFilePickerField from '../pickers/RemoteFilePickerField.svelte'

  let {
    sessionID = '',
    firstSessionID = '',
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  // Where to plant the hijack DLL on the target.
  let targetPath = $state('')
  // Where the legit DLL lives on the target so its exports can be cloned.
  let refPath = $state('')
  // Two exclusive ways to supply the hijack payload:
  //   'file'    — a fully-built DLL on operator disk
  //   'profile' — build one from an implant profile
  let sourceMode = $state('profile')
  let dllFile = $state('')
  let profile = $state('')
  // Optional: use a local reference DLL if the target one isn't accessible yet.
  let referenceFile = $state('')
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    targetPath = values['target'] || values['target path'] || ''
    refPath = values['reference-path'] || ''
    sourceMode = values['file'] ? 'file' : 'profile'
    dllFile = values['file'] || ''
    profile = values['profile'] || ''
    referenceFile = values['reference-file'] || ''
    timeout = values['timeout'] || ''
  }


  let cmdPreview = $derived.by(() => {
    const parts = ['dllhijack']
    if (refPath) parts.push('--reference-path', quote(refPath))
    if (referenceFile) parts.push('--reference-file', quote(referenceFile))
    if (sourceMode === 'file' && dllFile) parts.push('--file', quote(dllFile))
    else if (sourceMode === 'profile' && profile) parts.push('--profile', quote(profile))
    if (timeout) parts.push('--timeout', String(timeout))
    if (targetPath) parts.push(quote(targetPath))
    return parts.filter(Boolean).join(' ')
  })

  let canExecute = $derived.by(() => {
    if (!targetPath || !refPath) return false
    if (sourceMode === 'file') return !!dllFile
    return !!profile
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="DLL Hijack" size="2xl" {onclose}>
  
    <p class="text-fg-muted text-sm mb-4">
      Plant a malicious DLL alongside a legitimate binary so it's loaded instead of the real dependency.
      Sliver clones the reference DLL's exports so the target application still loads and runs.
    </p>

    <div class="mb-3">
      <RemoteFilePickerField
        bind:value={targetPath}
        sessionID={firstSessionID || sessionID}
        label="Target path (where to plant the DLL)"
        placeholder="C:\\Program Files\\App\\legit.dll"
        description="Full path on target — usually next to the vulnerable exe, matching the DLL search order"
        mode="file"
      />
    </div>

    <div class="mb-3">
      <RemoteFilePickerField
        bind:value={refPath}
        sessionID={firstSessionID || sessionID}
        label="Reference DLL on target"
        placeholder="C:\\Windows\\System32\\msasn1.dll"
        description="Legit DLL whose exports we clone so the process still loads normally"
        mode="file"
      />
    </div>

    <CollapsibleGroup title="Hijack DLL source" open={true}>
      <SelectField
        bind:value={sourceMode}
        label="How to supply the malicious DLL"
        options={[
          { value: 'profile', label: 'Build from implant profile' },
          { value: 'file', label: 'Local DLL file' },
        ]}
      />
      {#if sourceMode === 'file'}
        <FilePickerField bind:value={dllFile} label="Local hijack DLL" />
      {:else}
        <TextField
          bind:value={profile}
          label="Implant profile"
          placeholder="Name of a DLL-format implant profile"
          description="The profile must be built as a DLL (--format shared)"
        />
      {/if}
    </CollapsibleGroup>

    <CollapsibleGroup title="Advanced" open={false}>
      <FilePickerField bind:value={referenceFile} label="Reference DLL (local override)" />
      <p class="text-fg-muted text-xs mt-1 mb-2">Optional. Use if you already have the legit DLL on operator disk — Sliver skips downloading it from the target.</p>
      <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
    </CollapsibleGroup>

    <div class="mb-4">
      <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
      <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview}</code>
    </div>
  
  {#snippet footer()}
    <div class="flex justify-between items-center">
    <PresetPicker
      commandPath="dllhijack"
      currentValues={{
        'target': targetPath,
        'reference-path': refPath,
        'reference-file': referenceFile,
        'file': sourceMode === 'file' ? dllFile : '',
        'profile': sourceMode === 'profile' ? profile : '',
      }}
      onapply={(values) => {
        if (values['target'] != null) targetPath = values['target']
        if (values['reference-path'] != null) refPath = values['reference-path']
        if (values['reference-file'] != null) referenceFile = values['reference-file']
        if (values['file']) { dllFile = values['file']; sourceMode = 'file' }
        else if (values['profile']) { profile = values['profile']; sourceMode = 'profile' }
      }}
    />
    <div class="flex gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="primary" onclick={execute} disabled={!canExecute}>Plant</Button>
    </div>
  </div>
  {/snippet}
</Modal>
