<script>
  import Button from '$components/ui/Button.svelte'
  import IconButton from '$components/ui/IconButton.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import Select from '$components/ui/Select.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { monitorProviders } from '$stores/resources/monitorProviders.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(monitorProviders)
  import { MonitorStart, MonitorStop, addProvider, removeProvider } from '../../api/monitor.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'
  import { errorMessage } from '../../utils/errors.js'

  let { embedded = false, onclose } = $props()

  let adding = $state(false)
  let newID = $state('')
  let newType = $state('virustotal')
  let newAPIKey = $state('')
  let newAPIPassword = $state('')
  let actionError = $state('')

  let providers = $derived(monitorProviders.data || [])
  let providerRows = $derived(providers.map((provider, index) => ({
    _rowKey: provider.ID || provider.id || index,
    _id: provider.ID || provider.id || '-',
    _type: provider.Type || provider.type || '-',
  })))

  const PROVIDER_TYPES = [
    { value: 'virustotal', label: 'VirusTotal' },
    { value: 'xforce', label: 'IBM X-Force' },
    { value: 'greynoise', label: 'GreyNoise' },
    { value: 'otx', label: 'AlienVault OTX' },
  ]
  const columns = [
    { key: '_id', label: 'ID' },
    { key: '_type', label: 'Type' },
    { key: '_actions', label: 'Actions', width: 90, sortable: false },
  ]

  async function startGlobal() {
    actionError = ''
    try {
      await MonitorStart()
      await monitorProviders.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Failed to start monitoring: ')
    }
  }

  async function stopGlobal() {
    actionError = ''
    try {
      await MonitorStop()
      await monitorProviders.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Failed to stop monitoring: ')
    }
  }

  async function addNewProvider() {
    if (!newID || !newAPIKey) { actionError = 'ID and API key are required'; return }
    adding = true
    actionError = ''
    try {
      await addProvider(newID, newType, newAPIKey, newAPIPassword)
      newID = ''; newAPIKey = ''; newAPIPassword = ''
      await monitorProviders.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Add failed: ')
    } finally {
      adding = false
    }
  }

  async function deleteProvider(id) {
    if (!(await dialog.confirm(`Remove provider "${id}"?`, 'Confirm Remove'))) return
    actionError = ''
    try {
      await removeProvider(id)
      await monitorProviders.refresh()
    } catch (err) {
      actionError = errorMessage(err, 'Remove failed: ')
    }
  }
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Threat Monitors'} icon={embedded ? '' : 'shield'}>
  <Toolbar class="justify-end gap-2">
    <Button color="green" size="sm" onclick={startGlobal}>Start Monitoring</Button>
    <Button color="red" size="sm" onclick={stopGlobal}>Stop Monitoring</Button>
    <Button color="dark" size="sm" onclick={() => monitorProviders.refresh()} disabled={monitorProviders.loading}>
      Refresh
    </Button>
  </Toolbar>

  <div class="flex flex-1 min-h-0 flex-col gap-3 p-3 text-xs">
    {#if actionError}
      <div class="rounded border border-danger-500 bg-danger-500/10 p-2 text-danger-500">{actionError}</div>
    {/if}

    <section class="border border-line bg-panel rounded">
      <div class="border-b border-line px-3 py-2 font-semibold">Add Provider</div>
      <div class="grid grid-cols-1 gap-2 p-3 providers-grid">
        <TextInput size="sm" bind:value={newID} placeholder="Provider ID" class="font-mono min-w-0" />
        <Select bind:value={newType} options={PROVIDER_TYPES} aria-label="Provider type" class="min-w-0" />
        <TextInput size="sm" bind:value={newAPIKey} placeholder="API Key" class="font-mono min-w-0" />
        <TextInput size="sm" bind:value={newAPIPassword} placeholder="API Password (optional)" class="font-mono min-w-0" />
        <Button color="primary" size="sm" class="justify-self-start px-4" onclick={addNewProvider} disabled={adding}>
          {adding ? 'Adding...' : 'Add'}
        </Button>
      </div>
    </section>

    <div class="flex-1 min-h-0">
      <DataTable
        data={providerRows}
        {columns}
        keyField="_rowKey"
        loading={monitorProviders.loading}
        error={monitorProviders.error && !monitorProviders.loading ? monitorProviders.error : null}
        emptyState={{ icon: 'shield', title: 'No monitoring providers configured' }}
      >
        {#snippet children(provider, col)}
          {#if col.key === '_id'}
            <span class="font-mono">{provider._id}</span>
          {:else if col.key === '_actions'}
            <IconButton
              icon="trash"
              label="Remove"
              tooltip="Remove provider"
              color="red"
              size="xs"
              onclick={() => deleteProvider(provider._id)}
            />
          {:else}
            {provider[col.key]}
          {/if}
        {/snippet}
      </DataTable>
    </div>
  </div>
</Panel>
