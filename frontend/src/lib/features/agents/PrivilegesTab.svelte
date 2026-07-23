<script>
  import { onMount } from 'svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Button from '$components/ui/Button.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import ErrorState from '$components/ui/ErrorState.svelte'
  import { GetTokenPrivs, RevToSelfToken } from '../../api/token.js'
  import { errorMessage } from '../../utils/errors.js'
  import { dialog } from '../../stores/ui/dialog.svelte.js'

  let { sessionID = '' } = $props()

  let privData = $state(null)
  let loading = $state(false)
  let error = $state('')

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    privData = null
    try {
      privData = (await GetTokenPrivs(sessionID)) ?? null
    } catch (err) {
      error = errorMessage(err)
    } finally {
      loading = false
    }
  }

  async function revToSelf() {
    if (!await dialog.confirm('Revert to original process token? This will drop any impersonation.', 'Rev2Self')) return
    try {
      await RevToSelfToken(sessionID)
      await refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err), 'Rev2Self failed')
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

  let columns = [
    { key: '_name', label: 'Privilege', width: 240 },
    { key: '_description', label: 'Description' },
    { key: '_status', label: 'Status', width: 120 },
    { key: '_attributes', label: 'Attributes' },
  ]

  let rows = $derived((privData?.PrivInfo || []).map((entry, i) => ({
    _rowKey: entry.Name || i,
    _name: entry.Name || '-',
    _description: entry.Description || '-',
    _status: entry.Enabled ? 'Enabled' : 'Disabled',
    _enabled: entry.Enabled,
    _default: entry.EnabledByDefault || false,
    _removed: entry.Removed || false,
    _usedForAccess: entry.UsedForAccess || false,
    _attributes: [
      entry.EnabledByDefault ? 'Default' : '',
      entry.Removed ? 'Removed' : '',
      entry.UsedForAccess ? 'Used for Access' : '',
    ].filter(Boolean).join(', ') || '-',
  })))
</script>

<div class="flex flex-col h-full">
  <div class="tab-header flex items-center gap-2 py-2 pl-2">
    {#if privData}
      <span class="font-mono text-xs">{privData.ProcessName || 'Current Process'}</span>
      {#if integrityBadge.label}
        <Badge variant={integrityBadge.variant}>{integrityBadge.label}</Badge>
      {/if}
    {/if}
    <div class="flex-1"></div>
    <div class="flex items-center gap-2">
      <Button color="dark" size="xs" onclick={refresh} disabled={loading}>Refresh</Button>
      <Button color="warning" size="xs" onclick={revToSelf}>Rev2Self</Button>
    </div>
  </div>

  <div class="flex-1 overflow-y-auto">
    {#if error}
      <ErrorState {error} title="Failed to load privileges" class="m-2" />
    {/if}
    <DataTable
      data={rows}
      {columns}
      keyField="_rowKey"
      {loading}
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
</div>

