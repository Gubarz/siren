<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import { quote } from '../../../../utils/shell.js'
  import Button from '../../../../components/ui/Button.svelte'
  import TextField from '../../../../components/forms/TextField.svelte'
  import CheckboxField from '../../../../components/forms/CheckboxField.svelte'
  import SelectField from '../../../../components/forms/SelectField.svelte'
  import PresetPicker from '../../../../components/forms/PresetPicker.svelte'

  let {
    open = $bindable(false),
    onexecute,
    onclose,
    initialValues = {},
  } = $props()

  let kind = $state('tcp')
  let bind = $state('0.0.0.0')
  let lport = $state(9898)
  let allowAll = $state(false)

  $effect.pre(() => {
    resetForm(initialValues)
  })

  function resetForm(values) {
    kind = values['kind'] || 'tcp'
    bind = values['bind'] || '0.0.0.0'
    lport = values['lport'] || 9898
    allowAll = values['allow-all'] || false
  }


  let cmdPreview = $derived.by(() => {
    const parts = ['pivots', kind === 'named-pipe' ? 'named-pipe' : 'tcp']
    if (bind) parts.push('--bind', quote(bind))
    if (kind === 'tcp' && lport && Number(lport) !== 9898) parts.push('--lport', String(lport))
    if (kind === 'named-pipe' && allowAll) parts.push('--allow-all')
    return parts.join(' ')
  })

  function execute() {
    onexecute?.({ cmd: cmdPreview })
  }
</script>

<Modal bind:open title="Start Pivot Listener" size="xl" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    Open a pivot listener on the current implant so child implants can connect through it.
    TCP pivots listen on the implant's network interface; named-pipe pivots are Windows-only.
  </p>

  <div class="mb-3">
    <SelectField
      bind:value={kind}
      label="Pivot type"
      options={[
        { value: 'tcp', label: 'TCP' },
        { value: 'named-pipe', label: 'Named Pipe (Windows only)' },
      ]}
    />
  </div>

  {#if kind === 'tcp'}
    <div class="mb-3">
      <TextField
        bind:value={bind}
        label="Bind interface"
        placeholder="0.0.0.0"
        description="Remote interface on the implant to bind the TCP listener to"
      />
    </div>
    <div class="mb-3">
      <TextField
        bind:value={lport}
        label="Listener port"
        type="number"
        placeholder="9898"
      />
    </div>
  {:else}
    <div class="mb-3">
      <TextField
        bind:value={bind}
        label="Named pipe"
        placeholder="pipe/name"
        description="Named pipe name to bind the pivot listener to"
      />
    </div>
    <CheckboxField bind:checked={allowAll} label="Allow all users" description="Grant every local user access to the pipe" />
  {/if}

  <div class="mb-4">
    <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
    <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmdPreview}</code>
  </div>

  {#snippet footer()}
    <div class="flex justify-between items-center">
      <PresetPicker
        commandPath="pivots"
        currentValues={{ 'kind': kind, 'bind': bind, 'lport': lport, 'allow-all': allowAll }}
        onapply={(values) => {
          if (values['kind'] != null) kind = values['kind']
          if (values['bind'] != null) bind = values['bind']
          if (values['lport'] != null) lport = values['lport']
          if (values['allow-all'] != null) allowAll = values['allow-all']
        }}
      />
      <div class="flex gap-2">
        <Button color="dark" onclick={() => open = false}>Cancel</Button>
        <Button color="primary" onclick={execute} disabled={!bind}>Start</Button>
      </div>
    </div>
  {/snippet}
</Modal>
