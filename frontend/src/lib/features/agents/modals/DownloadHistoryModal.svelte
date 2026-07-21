<script>
  import IconButton from '$components/ui/IconButton.svelte'
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import Badge from '$components/ui/Badge.svelte'
  import DataTable from '$components/patterns/DataTable.svelte'
  import { GetDownloadHistory, GetAllDownloadHistory, ClearDownloadHistory } from '../../../api/agents.js'
  import { dialog } from '../../../stores/ui/dialog.svelte.js'

  let {
    isOpen = false,
    remotePath = '',
    sessionID = '',
    onclose,
  } = $props()

  let records = $state([])
  let loading = $state(false)
  let error = $state(null)
  let filterText = $state('')

  $effect(() => {
    if (isOpen) {
      loadHistory()
    }
  })

  async function loadHistory() {
    loading = true
    error = null
    try {
      if (remotePath || sessionID) {
        records = await GetDownloadHistory(sessionID, remotePath)
      } else {
        records = await GetAllDownloadHistory()
      }
    } catch (err) {
      error = err?.message || String(err)
      records = []
    } finally {
      loading = false
    }
  }

  async function handleClear() {
    const confirmMsg = remotePath
      ? `Clear download history for "${remotePath}"?`
      : 'Clear all download history?'
    if (!(await dialog.confirm(confirmMsg, 'Clear History'))) return
    try {
      await ClearDownloadHistory(sessionID, remotePath)
      await loadHistory()
    } catch (err) {
      await dialog.alert(`Failed to clear history: ${err?.message || err}`, 'Error')
    }
  }

  function formatSize(bytes) {
    if (!bytes || bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  let tableColumns = [
    { key: 'timestampStr', label: 'Time', width: 180 },
    { key: 'remotePath', label: 'Remote Path', width: 250 },
    { key: 'localPath', label: 'Local Destination', width: 250 },
    { key: 'sizeStr', label: 'Size', width: 90 },
    { key: 'status', label: 'Status', width: 110 },
    { key: 'error', label: 'Details' },
  ]

  let normalizedRows = $derived((records || []).map((r, idx) => ({
    ...r,
    _key: r.id || idx,
    timestampStr: r.timestamp ? new Date(r.timestamp).toLocaleString() : '-',
    sizeStr: formatSize(r.size),
  })))

  let filteredRows = $derived(
    !filterText
      ? normalizedRows
      : normalizedRows.filter((r) =>
          (r.remotePath || '').toLowerCase().includes(filterText.toLowerCase()) ||
          (r.localPath || '').toLowerCase().includes(filterText.toLowerCase()) ||
          (r.status || '').toLowerCase().includes(filterText.toLowerCase()) ||
          (r.error || '').toLowerCase().includes(filterText.toLowerCase())
        )
  )

  function getStatusVariant(status) {
    if (status === 'completed') return 'success'
    if (status === 'failed') return 'danger'
    return 'warning'
  }
</script>

{#if isOpen}
  <div
    class="fixed inset-0 bg-black/60 z-100 flex items-center justify-center p-4"
    onclick={onclose}
    onkeydown={(e) => { if (e.key === 'Escape') onclose() }}
    role="dialog"
    tabindex="-1"
  >
    <div
      class="w-full max-w-5xl h-vh-80 bg-panel border border-line rounded-lg flex flex-col overflow-hidden shadow-2xl"
      role="none"
      onclick={(e) => e.stopPropagation()}
    >
      <div class="flex items-center gap-3 px-4 py-3 bg-chrome border-b border-line">
        <div class="flex-1 min-w-0">
          <h2 class="text-base font-semibold text-fg m-0 flex items-center gap-2">
            <span>Download History</span>
            {#if remotePath}
              <span class="font-mono text-xs font-normal text-brand truncate max-w-md">({remotePath})</span>
            {/if}
          </h2>
        </div>
        <div class="w-48">
          <TextInput size="sm" placeholder="Filter history..." bind:value={filterText} class="font-mono" />
        </div>
        <Button color="dark" size="sm" onclick={loadHistory}>Refresh</Button>
        <Button color="red" size="sm" onclick={handleClear} disabled={records.length === 0}>Clear</Button>
        <IconButton icon="x" label="Close" tooltip="Close" onclick={onclose} />
      </div>

      <div class="flex-1 min-h-0 overflow-auto">
        <DataTable
          data={filteredRows}
          columns={tableColumns}
          keyField="_key"
          {loading}
          {error}
          emptyState={{ title: remotePath ? 'No download history for this file.' : 'No download history recorded.' }}
        >
          {#snippet children(item, col)}
            {#if col.key === 'timestampStr'}
              <span class="font-mono text-xs text-fg-muted">{item.timestampStr}</span>
            {:else if col.key === 'remotePath'}
              <span class="font-mono text-xs text-fg break-all">{item.remotePath}</span>
            {:else if col.key === 'localPath'}
              <span class="font-mono text-xs text-fg-muted break-all">{item.localPath}</span>
            {:else if col.key === 'sizeStr'}
              <span class="font-mono text-xs">{item.sizeStr}</span>
            {:else if col.key === 'status'}
              <Badge variant={getStatusVariant(item.status)} size="xs">
                {item.status}
              </Badge>
            {:else if col.key === 'error'}
              <span class="text-xs text-red-400 truncate">{item.error || '-'}</span>
            {:else}
              {item[col.key] ?? ''}
            {/if}
          {/snippet}
        </DataTable>
      </div>
    </div>
  </div>
{/if}
