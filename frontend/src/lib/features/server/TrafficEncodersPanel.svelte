<script>
  import Button from '$components/ui/Button.svelte'
  import Checkbox from '$components/ui/Checkbox.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import PresetPicker from '$components/forms/PresetPicker.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { trafficEncoders } from '$stores/resources/trafficEncoders.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(trafficEncoders)
  import { AddTrafficEncoder, RemoveTrafficEncoder } from '../../api/trafficEncoders.js'
  import { OpenFileDialog } from '../../api/runtime.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'
  import { formatBytes } from '../../utils/formats.js'

  let { embedded = false, onclose } = $props()

  let adding = $state(false)
  let skipTests = $state(false)
  let actionError = $state('')
  let testResult = $state(null)

  let encoders = $derived(trafficEncoders.data || [])
  let encoderRows = $derived(encoders.map((encoder, index) => ({
    _rowKey: encoder.id || encoder.name || index,
    _name: encoder.name || '-',
    _id: encoder.id || '-',
    _size: encoder.size || 0,
    _sizeText: formatBytes(encoder.size),
  })))
  let resultTests = $derived(testResult?.Tests || testResult?.tests || [])

  const columns = [
    { key: '_name', label: 'Name' },
    { key: '_id', label: 'ID' },
    { key: '_sizeText', label: 'Size', width: 120 },
    { key: '_actions', label: 'Actions', width: 90, sortable: false },
  ]

  async function addEncoder() {
    let path
    try {
      path = await OpenFileDialog('Add traffic encoder WASM')
    } catch (err) {
      actionError = errorMessage(err, 'Pick failed: ')
      return
    }
    if (!path) return
    adding = true
    actionError = ''
    testResult = null
    try {
      testResult = await AddTrafficEncoder(path, skipTests)
      await trafficEncoders.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Add failed: ')
    } finally {
      adding = false
    }
  }

  async function removeEncoder(name) {
    if (!(await dialog.confirm(`Remove traffic encoder "${name}"?`, 'Confirm Remove'))) return
    actionError = ''
    try {
      await RemoveTrafficEncoder(name)
      if (testResultName() === name) testResult = null
      await trafficEncoders.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Remove failed: ')
    }
  }

  function applyPreset(values) {
    if (values.skipTests != null) skipTests = Boolean(values.skipTests)
  }

  function testResultName() {
    const wasm = testResult?.Encoder?.Wasm || testResult?.encoder?.wasm || {}
    return wasm.Name || wasm.name || ''
  }

  function testPassed(test) {
    return Boolean(test?.Success ?? test?.success)
  }

  function testDuration(test) {
    const value = Number(test?.Duration ?? test?.duration ?? 0)
    if (!value) return '0 ms'
    return `${Math.round(value / 1000000)} ms`
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Traffic Encoders'} icon={embedded ? '' : 'shuffle'}>
  <Toolbar class="justify-end">
    <PresetPicker
      commandPath="traffic-encoders/add"
      currentValues={{ skipTests }}
      onapply={applyPreset}
    />
    <Checkbox bind:checked={skipTests} label="Skip tests" />
    <Button color="dark" size="sm" onclick={() => trafficEncoders.refresh()} disabled={trafficEncoders.loading}>
      Refresh
    </Button>
    <Button color="primary" size="sm" icon="plus" onclick={addEncoder} disabled={adding}>
      {adding ? 'Adding...' : 'Upload WASM'}
    </Button>
  </Toolbar>

  <div class="flex flex-1 min-h-0 flex-col gap-3 p-3 text-xs">
    {#if actionError}
      <div class="rounded border border-danger-500 bg-danger-500/10 p-2 text-danger-500">{actionError}</div>
    {/if}

    {#if testResult}
      <section class="border border-line bg-panel">
        <div class="border-b border-line px-3 py-2 font-semibold">
          Test results: <span class="font-mono">{testResultName()}</span>
        </div>
        <div class="divide-y divide-line">
          {#each resultTests as test}
            <div class="grid grid-cols-4 gap-2 px-3 py-2">
              <span class="font-mono">{test.Name || test.name}</span>
              <span class={testPassed(test) ? 'text-success-500' : 'text-danger-500'}>
                {testPassed(test) ? 'pass' : 'fail'}
              </span>
              <span class="font-mono text-fg-muted">{testDuration(test)}</span>
              <span class="truncate text-danger-500">{test.Err || test.err || ''}</span>
            </div>
          {:else}
            <div class="px-3 py-2 text-fg-muted">Tests skipped.</div>
          {/each}
        </div>
      </section>
    {/if}

    <div class="flex-1 min-h-0">
      <DataTable
        data={encoderRows}
        {columns}
        keyField="_rowKey"
        loading={trafficEncoders.loading}
        error={trafficEncoders.error && !trafficEncoders.loading ? trafficEncoders.error : null}
        emptyState={{ icon: 'shuffle', title: 'No traffic encoders installed' }}
      >
        {#snippet children(encoder, col)}
          {#if col.key === '_name' || col.key === '_id' || col.key === '_sizeText'}
            <span class="font-mono text-fg-muted">{encoder[col.key]}</span>
          {:else if col.key === '_actions'}
            <div class="flex justify-end">
              <IconButton
                icon="trash"
                label="Remove encoder"
                tooltip="Remove encoder"
                color="red"
                size="xs"
                onclick={() => removeEncoder(encoder._name)}
              />
            </div>
          {:else}
            {encoder[col.key]}
          {/if}
        {/snippet}
      </DataTable>
    </div>
  </div>
</Panel>
