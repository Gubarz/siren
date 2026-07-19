<script>
  import { onMount } from 'svelte'
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import PresetPicker from '$components/forms/PresetPicker.svelte'
  import PanelBody from '$components/patterns/PanelBody.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { RegisterExtensionFromPath, CallExtension, listExtensions, RegisterWasmExtensionFromPath, ExecWasmExtension, listWasmExtensions } from '../../api/extensions.js'
  import { OpenFileDialog } from '../../api/runtime.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'

  let { sessionID = '' } = $props()

  let extensions = $state([])
  let wasmExtensions = $state([])
  let loading = $state(false)
  let nativeTab = $state(true)
  let error = $state('')
  let callResult = $state('')
  let nativeName = $state('')
  let nativeTargetOS = $state('')
  let nativeInit = $state('')
  let wasmName = $state('')

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      const [exts, wasmExts] = await Promise.all([
        listExtensions(sessionID),
        listWasmExtensions(sessionID),
      ])
      extensions = exts || []
      wasmExtensions = wasmExts || []
    } catch (err) {
      error = errorMessage(err, 'Failed to load: ')
    } finally {
      loading = false
    }
  }

  async function registerNative() {
    let path
    try { path = await OpenFileDialog('Select DLL/extension') } catch { return }
    if (!path) return
    try {
      await RegisterExtensionFromPath(sessionID, nativeName, path, nativeTargetOS, nativeInit)
      await refresh()
    } catch (err) {
      error = errorMessage(err, 'Register failed: ')
    }
  }

  async function callNative(name) {
    const exportName = await dialog.prompt('Export function name:', 'Call Extension', '')
    if (!exportName) return
    const args = await dialog.prompt('Arguments (hex):', 'Call Extension Arguments', '')
    const argsBytes = args ? new Uint8Array(args.split(' ').map(h => parseInt(h, 16)).filter(n => !isNaN(n))) : new Uint8Array(0)
    try {
      const resp = await CallExtension(sessionID, name, exportName, Array.from(argsBytes), false)
      callResult = resp?.Output ? new TextDecoder().decode(new Uint8Array(resp.Output)) : '(empty response)'
    } catch (err) {
      error = errorMessage(err, 'Call failed: ')
    }
  }

  async function registerWasm() {
    let path
    try { path = await OpenFileDialog('Select WASM extension') } catch { return }
    if (!path) return
    try {
      await RegisterWasmExtensionFromPath(sessionID, wasmName, path)
      await refresh()
    } catch (err) {
      error = errorMessage(err, 'Register WASM failed: ')
    }
  }

  async function execWasm(name) {
    const args = await dialog.prompt('Arguments:', 'Exec WASM Extension', '')
    const argList = args ? args.split(/\s+/) : []
    try {
      const resp = await ExecWasmExtension(sessionID, name, argList, false)
      callResult = `Exit code: ${resp?.ExitCode ?? resp?.exitCode}\n\n${resp?.Stdout || resp?.stdout || ''}${resp?.Stderr || resp?.stderr ? '\n--- stderr ---\n' + (resp?.Stderr || resp?.stderr) : ''}`
    } catch (err) {
      error = errorMessage(err, 'Exec failed: ')
    }
  }

  function applyPreset(values) {
    if (values.nativeTab != null) nativeTab = Boolean(values.nativeTab)
    if (values.nativeName != null) nativeName = values.nativeName
    if (values.nativeTargetOS != null) nativeTargetOS = values.nativeTargetOS
    if (values.nativeInit != null) nativeInit = values.nativeInit
    if (values.wasmName != null) wasmName = values.wasmName
  }
</script>

<div class="flex flex-col h-full">
  <Toolbar class="justify-between">
    <div class="flex gap-1">
      <Button color={nativeTab ? 'primary' : 'dark'} size="xs" onclick={() => { nativeTab = true }}>Native/DLL</Button>
      <Button color={!nativeTab ? 'primary' : 'dark'} size="xs" onclick={() => { nativeTab = false }}>WASM</Button>
    </div>
    <div class="flex gap-1">
      <PresetPicker
        commandPath="extensions/register"
        currentValues={{ nativeTab, nativeName, nativeTargetOS, nativeInit, wasmName }}
        onapply={applyPreset}
      />
      {#if nativeTab}
        <Button color="primary" size="xs" icon="plus" onclick={registerNative}>Register DLL</Button>
      {:else}
        <Button color="primary" size="xs" icon="plus" onclick={registerWasm}>Register WASM</Button>
      {/if}
      <Button color="dark" size="xs" onclick={refresh} disabled={loading}>Refresh</Button>
    </div>
  </Toolbar>

  <div class="grid gap-2 border-b border-line p-2 text-xs md:grid-cols-3">
    {#if nativeTab}
      <TextInput size="xs" bind:value={nativeName} placeholder="Extension name (optional)" />
      <TextInput size="xs" bind:value={nativeTargetOS} placeholder="Target OS (optional)" />
      <TextInput size="xs" bind:value={nativeInit} placeholder="Init function (optional)" />
    {:else}
      <TextInput size="xs" bind:value={wasmName} placeholder="WASM name (optional)" />
    {/if}
  </div>

  <PanelBody error={error || null} empty={(!loading && !error && ((nativeTab && extensions.length === 0) || (!nativeTab && wasmExtensions.length === 0)))} emptyIcon="package" emptyTitle={nativeTab ? 'No extensions registered' : 'No WASM extensions registered'}>
    <div class="p-2">
      {#if callResult}
        <div class="mb-2 rounded border border-line bg-canvas p-2 text-xs">
          <div class="font-semibold mb-1">Output</div>
          <pre class="whitespace-pre-wrap break-all">{callResult}</pre>
        </div>
      {/if}
      <table class="w-full border-collapse text-xs">
        <thead>
          <tr class="border-b border-line bg-table-header text-left text-fg-muted">
            <th class="px-3 py-2 font-medium">Name</th>
            <th class="px-3 py-2 text-right font-medium">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each (nativeTab ? extensions : wasmExtensions) as name}
            <tr class="border-b border-line hover:bg-row-hover">
              <td class="px-3 py-2 font-mono">{name}</td>
              <td class="px-3 py-2 text-right">
                {#if nativeTab}
                  <Button color="primary" size="xs" onclick={() => callNative(name)}>Call</Button>
                {:else}
                  <Button color="primary" size="xs" onclick={() => execWasm(name)}>Exec</Button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </PanelBody>
</div>
