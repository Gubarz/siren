<script>
  import Button from '$components/ui/Button.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import PanelBody from '$components/patterns/PanelBody.svelte'
  import FilePickerField from '$components/forms/FilePickerField.svelte'
  import SelectField from '$components/forms/SelectField.svelte'
  import TextField from '$components/forms/TextField.svelte'
  import { shellcodeEncoders } from '$stores/resources/shellcodeEncoders.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(shellcodeEncoders)
  import { EncodeShellcode, GenerateShellcodeRDI } from '../../api/shellcode.js'
  import { errorMessage } from '../../utils/errors.js'

  let { embedded = false, onclose } = $props()

  let rdiPath = $state('')
  let rdiFunction = $state('')
  let rdiArgs = $state('')
  let rdiBusy = $state(false)
  let rdiStatus = $state('')
  let rdiError = $state('')

  let shellcodePath = $state('')
  let architecture = $state('amd64')
  let encoder = $state('')
  let iterations = $state(1)
  let badCharsHex = $state('')
  let encodeBusy = $state(false)
  let encodeStatus = $state('')
  let encodeError = $state('')

  let encoders = $derived(shellcodeEncoders.data || [])
  let archOptions = $derived(
    [...new Set(encoders.map((item) => item.arch))]
      .map((arch) => ({ value: arch, label: arch })),
  )
  let encoderOptions = $derived(
    encoders
      .filter((item) => item.arch === architecture)
      .map((item) => ({ value: String(item.value), label: item.name })),
  )

  $effect(() => {
    if (archOptions.length > 0 && !archOptions.some((item) => item.value === architecture)) {
      architecture = archOptions[0].value
    }
  })

  $effect(() => {
    if (encoderOptions.length > 0 && !encoderOptions.some((item) => item.value === encoder)) {
      encoder = encoderOptions[0].value
    }
  })

  async function generateRDI() {
    if (!rdiPath) return
    rdiBusy = true
    rdiError = ''
    rdiStatus = ''
    try {
      const path = await GenerateShellcodeRDI({
        localPath: rdiPath,
        functionName: rdiFunction,
        arguments: rdiArgs,
      })
      rdiStatus = path ? `Saved to ${path}` : 'Cancelled.'
    } catch (err) {
      rdiError = errorMessage(err, 'RDI failed: ')
    } finally {
      rdiBusy = false
    }
  }

  async function encodeShellcode() {
    if (!shellcodePath || !encoder) return
    encodeBusy = true
    encodeError = ''
    encodeStatus = ''
    try {
      const path = await EncodeShellcode({
        localPath: shellcodePath,
        encoder: Number(encoder),
        architecture,
        iterations: Number(iterations) || 1,
        badCharsHex,
      })
      encodeStatus = path ? `Saved to ${path}` : 'Cancelled.'
    } catch (err) {
      encodeError = errorMessage(err, 'Encode failed: ')
    } finally {
      encodeBusy = false
    }
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Shellcode'} icon={embedded ? '' : 'binary'}>
  <PanelBody error={shellcodeEncoders.error && !shellcodeEncoders.loading ? shellcodeEncoders.error : null}>
    <div class="grid gap-4 p-3 text-sm">
      <section class="border border-line bg-panel p-3">
        <h3 class="mb-3 text-sm font-semibold text-fg">DLL to RDI Shellcode</h3>
        <FilePickerField bind:value={rdiPath} label="DLL file" accept=".dll" />
        <div class="grid grid-cols-2 gap-3">
          <TextField bind:value={rdiFunction} label="Export function" placeholder="Optional" />
          <TextField bind:value={rdiArgs} label="Arguments" placeholder="Optional" />
        </div>
        <div class="mt-2 flex items-center gap-3">
          <Button color="primary" size="sm" onclick={generateRDI} disabled={!rdiPath || rdiBusy}>
            {rdiBusy ? 'Generating...' : 'Generate RDI'}
          </Button>
          {#if rdiStatus}<span class="break-all text-xs text-success-500">{rdiStatus}</span>{/if}
          {#if rdiError}<span class="text-xs text-danger-500">{rdiError}</span>{/if}
        </div>
      </section>

      <section class="border border-line bg-panel p-3">
        <h3 class="mb-3 text-sm font-semibold text-fg">Encode Shellcode</h3>
        <FilePickerField bind:value={shellcodePath} label="Shellcode file" />
        <div class="grid grid-cols-3 gap-3">
          <SelectField bind:value={architecture} label="Architecture" options={archOptions} />
          <SelectField bind:value={encoder} label="Encoder" options={encoderOptions} disabled={encoderOptions.length === 0} />
          <TextField bind:value={iterations} label="Iterations" type="number" />
        </div>
        <TextField bind:value={badCharsHex} label="Bad chars" placeholder="000a0d" />
        <div class="mt-2 flex items-center gap-3">
          <Button color="primary" size="sm" onclick={encodeShellcode} disabled={!shellcodePath || !encoder || encodeBusy}>
            {encodeBusy ? 'Encoding...' : 'Encode'}
          </Button>
          <Button color="dark" size="sm" onclick={() => shellcodeEncoders.refresh()} disabled={shellcodeEncoders.loading}>
            Refresh Encoders
          </Button>
          {#if encodeStatus}<span class="break-all text-xs text-success-500">{encodeStatus}</span>{/if}
          {#if encodeError}<span class="text-xs text-danger-500">{encodeError}</span>{/if}
        </div>
      </section>
    </div>
  </PanelBody>
</Panel>
