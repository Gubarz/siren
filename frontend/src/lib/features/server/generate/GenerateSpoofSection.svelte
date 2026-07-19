<script>
  import Button from '$components/ui/Button.svelte'
  import TextField from '$components/forms/TextField.svelte'
  import CollapsibleGroup from '$components/forms/CollapsibleGroup.svelte'
  import { OpenFileDialog } from '../../../api/runtime.js'
  import { PrimeSpoofMetadataFromPath } from '../../../api/operatorControls.js'
  import { errorMessage } from '../../../utils/errors.js'

  // Optional PE-spoof-metadata preflight. Sliver's server keeps spoof
  // metadata in a per-implant-name map; we prime it here so the following
  // Generate call picks it up automatically. Skippable — most builds
  // don't need it.

  let { implantName = '', goos = 'windows' } = $props()

  let sourcePath = $state('')
  let priming = $state(false)
  let status = $state('')

  async function pickPE() {
    try {
      const path = await OpenFileDialog('Select reference PE for spoof metadata')
      if (path) sourcePath = path
    } catch (err) {
      status = errorMessage(err, 'File dialog failed: ')
    }
  }

  async function prime() {
    if (!implantName || !sourcePath) return
    priming = true
    status = ''
    try {
      await PrimeSpoofMetadataFromPath(implantName, sourcePath)
      status = 'Spoof metadata staged — hit Generate to build the implant against it.'
    } catch (err) {
      status = errorMessage(err, 'Prime failed: ')
    } finally {
      priming = false
    }
  }
</script>

{#if goos === 'windows'}
  <CollapsibleGroup title="Spoof PE metadata (advanced)" open={false}>
    <p class="text-xs text-fg-muted mb-2">
      Clone the version-info / icon / resource directory from an existing PE
      onto the implant binary. Requires a non-empty implant name so the
      server can match this metadata to the follow-up Generate call.
    </p>
    <div class="flex items-end gap-2">
      <div class="flex-1">
        <TextField
          bind:value={sourcePath}
          label="Reference PE path"
          placeholder="C:\Windows\System32\notepad.exe"
        />
      </div>
      <Button color="dark" size="sm" onclick={pickPE}>Browse…</Button>
      <Button color="primary" size="sm" onclick={prime} disabled={priming || !implantName || !sourcePath}>
        {priming ? 'Priming…' : 'Prime metadata'}
      </Button>
    </div>
    {#if !implantName}
      <p class="mt-1 text-xs text-warning-500">Set an implant name at the top of the form first.</p>
    {/if}
    {#if status}
      <p class="mt-2 text-xs text-fg-muted">{status}</p>
    {/if}
  </CollapsibleGroup>
{/if}
