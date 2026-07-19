<script>
  import { pivots } from '$stores/resources/pivots.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(pivots)
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'

  let {
    embedded = false,
    onclose,
  } = $props()

  function flattenGraph(nodes, depth = 0, result = []) {
    if (!nodes) return result
    for (const node of nodes) {
      result.push({ ...node, _depth: depth })
      flattenGraph(node.Children || node.children || [], depth + 1, result)
    }
    return result
  }

  let flatNodes = $derived(flattenGraph(pivots.data?.Children || pivots.data?.children || []))
  let pivotRows = $derived(flatNodes.map((node, index) => ({
    _rowKey: node.PeerID || node.peerID || node.Session?.ID || node.session?.id || index,
    _depth: node._depth || 0,
    _tree: node._depth === 0 ? 'root' : 'peer',
    _name: node.Name || node.name || '-',
    _peerID: node.PeerID || node.peerID || '-',
    _sessionID: (node.Session?.ID || node.session?.id || '-').substring(0, 8),
  })))

  const columns = [
    { key: '_tree', label: 'Tree', width: 120 },
    { key: '_name', label: 'Name' },
    { key: '_peerID', label: 'Peer ID' },
    { key: '_sessionID', label: 'Session ID', width: 120 },
  ]
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Pivots'} icon={embedded ? '' : 'network'}>
  <div class="flex-1 min-h-0">
    <DataTable
      data={pivotRows}
      {columns}
      keyField="_rowKey"
      loading={pivots.loading}
      error={pivots.error && !pivots.loading ? pivots.error : null}
      emptyState={{ icon: 'network', title: 'No active pivots' }}
    >
      {#snippet children(node, col)}
        {#if col.key === '_tree'}
          <span class="font-mono">
            <span style="display:inline-block; width: {node._depth * 20}px"></span>
            {#if node._depth > 0}\u21B3{/if}{node._tree}
          </span>
        {:else if col.key === '_peerID' || col.key === '_sessionID'}
          <span class="font-mono">{node[col.key]}</span>
        {:else}
          {node[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</Panel>
