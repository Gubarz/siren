<script>
  import Button from '$components/ui/Button.svelte'
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Toolbar from '$components/patterns/Toolbar.svelte'
  import { builders } from '$stores/resources/builders.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(builders)

  let { embedded = false, onclose } = $props()

  let builderList = $derived(builders.data || [])
  let builderRows = $derived(builderList.map((builder, index) => ({
    _rowKey: builder.ID || builder.id || builder.Name || builder.name || index,
    _name: builder.Name || builder.name || '-',
    _operator: builder.OperatorName || builder.operatorName || '-',
    _osArch: `${builder.GOOS || builder.goos || '-'}/${builder.GOARCH || builder.goarch || '-'}`,
    _templates: (builder.Templates || builder.templates || []).join(', ') || '-',
    _targets: (builder.Targets || builder.targets || []).length,
  })))

  const columns = [
    { key: '_name', label: 'Name' },
    { key: '_operator', label: 'Operator' },
    { key: '_osArch', label: 'OS/Arch' },
    { key: '_templates', label: 'Templates' },
    { key: '_targets', label: 'Targets', width: 90 },
  ]
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Build Farm'} icon={embedded ? '' : 'hammer'}>
  <Toolbar class="justify-end">
    <Button color="dark" size="sm" onclick={() => builders.refresh()} disabled={builders.loading}>
      Refresh
    </Button>
  </Toolbar>

  <div class="flex-1 min-h-0">
    <DataTable
      data={builderRows}
      {columns}
      keyField="_rowKey"
      loading={builders.loading}
      error={builders.error && !builders.loading ? builders.error : null}
      emptyState={{ icon: 'hammer', title: 'No registered builders' }}
    >
      {#snippet children(builder, col)}
        {#if col.key === '_name' || col.key === '_osArch' || col.key === '_templates' || col.key === '_targets'}
          <span class="font-mono text-fg-muted">{builder[col.key]}</span>
        {:else}
          {builder[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
