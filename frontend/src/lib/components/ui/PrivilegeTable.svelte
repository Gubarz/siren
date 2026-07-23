<script>
  import DataTable from '$components/patterns/DataTable.svelte'
  import Badge from '$components/ui/Badge.svelte'

  let { privData = null, loading = false } = $props()

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
