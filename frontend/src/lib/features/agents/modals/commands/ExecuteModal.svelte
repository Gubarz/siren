<script>
  import CommandModal from '../../../../components/patterns/CommandModal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import CollapsibleGroup from '../../../../components/forms/CollapsibleGroup.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
  import PidPickerField from '../pickers/PidPickerField.svelte'
  import MemfilePickerButton from '../pickers/MemfilePickerButton.svelte'

  let {
    firstSessionID = '',
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let program = $state('')
  let programArgs = $state('')
  let output = $state(true)
  let ignoreStderr = $state(false)
  let stdoutFile = $state('')
  let stderrFile = $state('')
  let ppid = $state(0)
  let useToken = $state(false)
  let saveToLoot = $state(false)
  let lootName = $state('')
  let timeout = $state('')

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    program = values['command'] || values['program'] || ''
    programArgs = values['arguments'] || ''
    output = values['output'] ?? true
    ignoreStderr = values['ignore-stderr'] || false
    stdoutFile = values['stdout'] || ''
    stderrFile = values['stderr'] || ''
    ppid = values['ppid'] || 0
    useToken = values['token'] || false
    saveToLoot = values['loot'] || false
    lootName = values['name'] || ''
    timeout = values['timeout'] || ''
  }

  let cmdPreview = $derived.by(() => {
    const parts = ['execute']
    if (output) parts.push('--output')
    if (ignoreStderr) parts.push('--ignore-stderr')
    if (stdoutFile) parts.push('--stdout', quote(stdoutFile))
    if (stderrFile) parts.push('--stderr', quote(stderrFile))
    if (ppid) parts.push('--ppid', String(ppid))
    if (useToken) parts.push('--token')
    if (saveToLoot) {
      parts.push('--loot')
      if (lootName) parts.push('--name', quote(lootName))
    }
    if (timeout) parts.push('--timeout', String(timeout))
    if (program) parts.push(quote(program))
    if (programArgs) parts.push(programArgs)
    return parts.filter(Boolean).join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<CommandModal
  bind:open
  title="Execute Program"
  size="2xl"
  description="Run a command on the remote system and (optionally) capture its output."
  {cmdPreview}
  presetPath="execute"
  presetValues={{
    'command': program,
    'arguments': programArgs,
    'output': output,
    'ignore-stderr': ignoreStderr,
    'stdout': stdoutFile,
    'stderr': stderrFile,
    'ppid': ppid,
    'token': useToken,
    'loot': saveToLoot,
    'name': lootName,
  }}
  onpresetapply={(values) => {
    if (values['command'] != null) program = values['command']
    if (values['arguments'] != null) programArgs = values['arguments']
    if (values['output'] != null) output = values['output']
    if (values['ignore-stderr'] != null) ignoreStderr = values['ignore-stderr']
    if (values['stdout'] != null) stdoutFile = values['stdout']
    if (values['stderr'] != null) stderrFile = values['stderr']
    if (values['ppid'] != null) ppid = values['ppid']
    if (values['token'] != null) useToken = values['token']
    if (values['loot'] != null) saveToLoot = values['loot']
    if (values['name'] != null) lootName = values['name']
  }}
  onexecute={execute}
  disabled={!program}
  {onclose}
>
  <div class="mb-3">
    <TextField
      bind:value={program}
      label="Program"
      placeholder="whoami, cmd.exe, /bin/bash, powershell.exe, etc."
      description="Absolute path or program on PATH — or pick a registered memfile below"
    />
    <div class="mt-1">
      <MemfilePickerButton
        sessionID={firstSessionID}
        label="Use memfile…"
        onpick={(path) => program = path}
      />
    </div>
  </div>

  <div class="mb-3">
    <TextField
      bind:value={programArgs}
      label="Arguments"
      placeholder="Arguments to pass"
    />
  </div>

  <CollapsibleGroup title="Output" open={true}>
    <CheckboxField bind:checked={output} label="Capture output" description="Return stdout/stderr back to the operator" />
    <CheckboxField bind:checked={ignoreStderr} label="Ignore stderr" description="Discard stderr on the target instead of capturing it" />
    <TextField
      bind:value={stdoutFile}
      label="Remote stdout file"
      placeholder="Path on target to redirect stdout"
    />
    <TextField
      bind:value={stderrFile}
      label="Remote stderr file"
      placeholder="Path on target to redirect stderr"
    />
    <CheckboxField bind:checked={saveToLoot} label="Save output to loot store" />
    {#if saveToLoot}
      <TextField bind:value={lootName} label="Loot entry name" placeholder="Default: program name" />
    {/if}
  </CollapsibleGroup>

  <CollapsibleGroup title="Process (advanced)" open={false}>
    <PidPickerField bind:value={ppid} label="Parent PID (spoof)" sessionID={firstSessionID} />
    <CheckboxField bind:checked={useToken} label="Use current impersonation token" description="Spawn under the token currently held by the implant" />
    <TextField bind:value={timeout} label="Timeout (seconds)" type="number" />
  </CollapsibleGroup>
</CommandModal>
