<script>
  import DataTable from '$components/patterns/DataTable.svelte'
  import Button from '$components/ui/Button.svelte'
  import { eventLog } from '$stores/resources/events.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(eventLog)

  function fmtTime(ts) {
    if (!ts) return "-";
    return new Date(ts).toLocaleTimeString();
  }

  let eventRows = $derived(eventLog.events.slice().reverse().map((event, index) => ({
    ...event,
    _rowKey: `${event.time || ''}:${event.type || ''}:${index}`,
    _time: fmtTime(event.time),
  })))
  let isLoading = $derived(eventLog.loading)
  let canLoadMore = $derived(eventLog.hasMore)

  const columns = [
    { key: '_time', label: 'Time', width: 100 },
    { key: 'type', label: 'Type', width: 160 },
    { key: '_details', label: 'Details', sortable: false },
  ]
</script>

<div class="flex h-full min-h-0 flex-col overflow-hidden">
  <div class="flex shrink-0 items-center gap-2 border-b border-line bg-chrome px-3 py-2 text-xs text-fg-muted">
    <span class="font-mono">{eventRows.length} event{eventRows.length === 1 ? '' : 's'}</span>
    <div class="ml-auto flex items-center gap-2">
      <Button
        color="dark"
        size="xs"
        icon="refresh"
        loading={isLoading}
        onclick={() => eventLog.refresh()}
      >
        Refresh
      </Button>
      <Button
        color="alternative"
        size="xs"
        icon="history"
        disabled={!canLoadMore || isLoading}
        onclick={() => eventLog.loadMore()}
      >
        Older
      </Button>
    </div>
  </div>
  <div class="min-h-0 flex-1 overflow-hidden">
    <DataTable
      data={eventRows}
      {columns}
      keyField="_rowKey"
      emptyState={{ icon: 'history', title: 'No events recorded yet' }}
    >
      {#snippet children(event, col)}
        {#if col.key === '_time' || col.key === 'type'}
          <span class="font-mono">{event[col.key]}</span>
        {:else if col.key === '_details'}
          <span class="font-mono">
            {#if event.sessionID}
              [Session {event.sessionID.substring(0,8)}] {event.username}@{event.hostname}
            {/if}
            {#if event.job}
              [Job {event.job}]
            {/if}
            {#if event.data}
              {event.data}
            {/if}
          </span>
        {:else}
          {event[col.key]}
        {/if}
      {/snippet}
    </DataTable>
  </div>
</div>
