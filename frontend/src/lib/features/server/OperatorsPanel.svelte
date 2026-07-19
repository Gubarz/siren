<script>
  import { operators } from '$stores/resources/operators.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(operators)
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Badge from '$components/ui/Badge.svelte'

  let {
    embedded = false,
    onclose,
  } = $props()

  let operatorRows = $derived((operators.data || []).map((operator, index) => ({
    _rowKey: operator.ID || operator.id || operator.Name || operator.name || index,
    _online: Boolean(operator.Online || operator.online),
    _name: operator.Name || operator.name || '-',
  })))

  const columns = [
    { key: '_online', label: 'Status', width: 90 },
    { key: '_name', label: 'Name' },
  ]
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Operators'} icon={embedded ? '' : 'users'}>
  <div class="flex-1 min-h-0">
    <DataTable
      data={operatorRows}
      {columns}
      keyField="_rowKey"
      loading={operators.loading}
      error={operators.error && !operators.loading ? operators.error : null}
      emptyState={{ icon: 'users', title: 'No operators connected' }}
    >
      {#snippet children(operator, col)}
        {#if col.key === '_online'}
          {#if operator._online}
            <Badge variant="success">Online</Badge>
          {:else}
            <Badge variant="danger">Offline</Badge>
          {/if}
        {:else if col.key === '_name'}
          <span class="font-mono">{operator._name}</span>
        {:else}
          {operator[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
