<script>
  import { onMount } from 'svelte'
  import Modal from '$components/patterns/Modal.svelte'
  import Button from '$components/ui/Button.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import { GetTokenPrivs, RevToSelfToken } from '../../../../api/token.js'
  import { errorMessage } from '../../../../utils/errors.js'

  let {
    open = $bindable(false),
    firstSessionID = '',
    onclose,
  } = $props()

  let privData = $state(null)
  let loading = $state(false)
  let error = $state('')

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      privData = (await GetTokenPrivs(firstSessionID)) ?? null
    } catch (err) {
      error = errorMessage(err)
    } finally {
      loading = false
    }
  }

  async function revToSelf() {
    try {
      await RevToSelfToken(firstSessionID)
      await refresh()
    } catch (err) {
      error = errorMessage(err, 'Rev2Self failed: ')
    }
  }

  let integrityBadge = $derived.by(() => {
    const level = privData?.ProcessIntegrity || ''
    const l = level.toLowerCase()
    if (l.includes('system')) return { variant: 'danger', label: level }
    if (l.includes('high')) return { variant: 'warning', label: level }
    if (l.includes('medium')) return { variant: 'info', label: level }
    if (l.includes('low')) return { variant: 'secondary', label: level }
    return { variant: 'secondary', label: level || 'Unknown' }
  })

  let summary = $derived.by(() => {
    if (!privData) return null
    const total = privData.PrivInfo?.length || 0
    const enabled = privData.PrivInfo?.filter((e) => e.Enabled)?.length || 0
    return { total, enabled }
  })

  let columns = [
    { key: '_name', label: 'Privilege', width: 200 },
    { key: '_description', label: 'Description' },
    { key: '_status', label: 'Status', width: 100 },
    { key: '_attributes', label: 'Attributes' },
  ]

  let rows = $derived((privData?.PrivInfo || []).map((entry, i) => ({
    _rowKey: entry.Name || i,
    _name: entry.Name || '-',
    _description: entry.Description || '-',
    _status: entry.Enabled ? 'Enabled' : 'Disabled',
    _enabled: entry.Enabled,
    _attributes: [
      entry.EnabledByDefault ? 'Default' : '',
      entry.Removed ? 'Removed' : '',
      entry.UsedForAccess ? 'Used for Access' : '',
    ].filter(Boolean).join(', ') || '-',
  })))
</script>

<Modal bind:open title="Get Privileges" size="3xl" {onclose}>
  {#if loading}
    <p class="text-fg-muted text-sm">Loading privileges...</p>
  {:else if error}
    <p class="text-danger-500 text-sm mb-4">{error}</p>
    <div class="flex justify-end gap-2">
      <Button color="dark" onclick={refresh}>Retry</Button>
    </div>
  {:else if privData}
    <div class="flex items-center gap-2 mb-4">
      <span class="text-sm font-mono">{privData.ProcessName || 'Current Process'}</span>
      {#if integrityBadge.label}
        <Badge variant={integrityBadge.variant}>{integrityBadge.label}</Badge>
      {/if}
      {#if summary}
        <span class="text-xs text-fg-muted">{summary.enabled}/{summary.total} enabled</span>
      {/if}
    </div>

    <div class="max-h-96 overflow-auto">
      <DataTable
        data={rows}
        {columns}
        keyField="_rowKey"
        emptyState={{ title: 'No privilege information available.' }}
      >
        {#snippet children(row, col)}
          {#if col.key === '_status'}
            {#if row._enabled}
              <Badge variant="success">Enabled</Badge>
            {:else}
              <Badge variant="secondary">Disabled</Badge>
            {/if}
          {:else if col.key === '_name'}
            <code class="text-xs font-mono">{row._name}</code>
          {:else if col.key === '_attributes'}
            <span class="text-xs text-fg-muted">{row._attributes}</span>
          {:else}
            <span class="text-xs">{row[col.key]}</span>
          {/if}
        {/snippet}
      </DataTable>
    </div>
  {:else}
    <p class="text-fg-muted text-sm">No data received.</p>
  {/if}

  {#snippet footer()}
    <div class="flex justify-between items-center">
      <Button color="warning" size="xs" onclick={revToSelf}>Rev2Self</Button>
      <div class="flex gap-2">
        {#if privData}
          <Button color="dark" size="xs" onclick={refresh}>Refresh</Button>
        {/if}
        <Button color="dark" onclick={() => open = false}>Close</Button>
      </div>
    </div>
  {/snippet}
</Modal>
