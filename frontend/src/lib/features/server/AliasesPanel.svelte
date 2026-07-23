<script>
  import { onMount } from 'svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import { GetAliases } from '../../api/aliases.js'
  import { errorMessage } from '../../utils/errors.js'

  let { embedded = false, onclose } = $props()

  let items = $state([])
  let loading = $state(false)
  let error = $state('')

  onMount(() => refresh())

  async function refresh() {
    loading = true
    error = ''
    try {
      items = (await GetAliases()) || []
    } catch (err) {
      error = errorMessage(err)
    } finally {
      loading = false
    }
  }

  let rows = $derived(items.map((item, i) => ({
    _rowKey: `${item.commandName ?? ''}-${item.name ?? ''}-${i}`,
    _name: item.name || '-',
    _commandName: item.commandName || '-',
    _version: item.version || '-',
    _type: item.type || 'alias',
    _help: item.help || '',
  })))

  const columns = [
    { key: '_type', label: 'Type', width: 100 },
    { key: '_name', label: 'Name', width: 160 },
    { key: '_commandName', label: 'Command', width: 180 },
    { key: '_version', label: 'Version', width: 90 },
    { key: '_help', label: 'Description' },
  ]
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Aliases & Extensions'}     icon={embedded ? '' : 'package'}
>
  <div class="flex-1 min-h-0">
    <DataTable
      data={rows}
      {columns}
      keyField="_rowKey"
      {loading}
      error={error || null}
      emptyState={{ icon: 'package', title: 'No aliases or extensions installed' }}
    >
      {#snippet children(row, col)}
        {#if col.key === '_type'}
          {#if row._type === 'extension'}
            <Badge variant="info">Extension</Badge>
          {:else}
            <Badge variant="secondary">Alias</Badge>
          {/if}
        {:else if col.key === '_name'}
          <span class="font-mono text-xs">{row._name}</span>
        {:else if col.key === '_commandName'}
          <code class="text-xs font-mono">{row._commandName}</code>
        {:else if col.key === '_help'}
          <span class="text-xs text-fg-muted line-clamp-2">{row._help}</span>
        {:else}
          <span class="text-xs">{row[col.key]}</span>
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
