<script>
  import { pivots } from '$stores/resources/pivots.svelte.js'
  import { pivotListeners } from '$stores/resources/pivotListeners.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'
  import { PivotStopListener } from '../../api/server.js'

  useResource(pivots, pivotListeners)
  import Panel from '$components/patterns/Panel.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import Button from '$components/ui/Button.svelte'

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

  let listenerIdCounter = 0
  let listenerRows = $derived((pivotListeners.data || []).map((listener) => {
    const fullSessionID = listener.ParentSessionID ?? listener.parentSessionID ?? '-'
    return {
      _rowKey: `${fullSessionID}-${listener.ID || listener.id}-${listenerIdCounter++}`,
      _id: listener.ID ?? listener.id ?? '-',
      _sessionID: fullSessionID.substring(0, 8),
      _fullSessionID: fullSessionID,
      _type: listener.Type ?? listener.type ?? '-',
      _bindAddress: listener.BindAddress ?? listener.bindAddress ?? '-',
      _pivotCount: (listener.Pivots ?? listener.pivots ?? []).length,
    }
  }))

  async function stopListener(sessionID, id) {
    try {
      await PivotStopListener(sessionID, id)
      await pivotListeners.refresh()
    } catch {}
  }

  const graphColumns = [
    { key: '_tree', label: 'Tree', width: 120 },
    { key: '_name', label: 'Name' },
    { key: '_peerID', label: 'Peer ID' },
    { key: '_sessionID', label: 'Session ID', width: 120 },
  ]

  const listenerColumns = [
    { key: '_id', label: 'ID', width: 60 },
    { key: '_type', label: 'Type', width: 120 },
    { key: '_bindAddress', label: 'Bind Address' },
    { key: '_sessionID', label: 'Session ID', width: 120 },
    { key: '_pivotCount', label: 'Pivots', width: 80 },
    { key: '_actions', label: '', width: 80, sortable: false },
  ]
</script>

<Panel {embedded} {onclose} title={embedded ? '' : 'Pivots'} icon={embedded ? '' : 'network'}>
  <div class="flex-1 min-h-0 flex flex-col gap-1">
    <div class="flex-1 min-h-0">
      <DataTable
        data={pivotRows}
        columns={graphColumns}
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

    <div class="flex-1 min-h-0">
      <div class="px-3 py-1 text-xs font-semibold text-fg-muted border-t border-line">Pivot Listeners</div>
      <DataTable
        data={listenerRows}
        columns={listenerColumns}
        keyField="_rowKey"
        loading={pivotListeners.loading}
        error={pivotListeners.error && !pivotListeners.loading ? pivotListeners.error : null}
        emptyState={{ icon: 'headphones', title: 'No active pivot listeners' }}
      >
        {#snippet children(listener, col)}
          {#if col.key === '_id' || col.key === '_sessionID'}
            <span class="font-mono">{listener[col.key]}</span>
          {:else if col.key === '_type'}
            <span class="font-mono text-xs bg-surface px-1 rounded">{listener._type}</span>
          {:else if col.key === '_pivotCount'}
            <span class="font-mono">{listener._pivotCount}</span>
          {:else if col.key === '_actions'}
            <div class="flex justify-end">
              <Button color="red" size="xs" onclick={() => stopListener(listener._fullSessionID, listener._id)}>Stop</Button>
            </div>
          {:else}
            {listener[col.key]}
          {/if}
        {/snippet}
      </DataTable>
    </div>
  </div>
</Panel>
