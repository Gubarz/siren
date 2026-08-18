<script>
  import Badge from '$components/ui/Badge.svelte'
  import EmptyState from '$components/ui/EmptyState.svelte'
  import { formatDateTime } from '../../utils/formats.js'

  let {
    selectedRun = null,
  } = $props()

  let output = $derived(
    selectedRun?.output || (selectedRun?.status === 'running' ? 'Running...' : 'No command output.'),
  )
</script>

<div class="flex h-full min-h-0 flex-col bg-canvas">
  {#if selectedRun}
    <header class="flex shrink-0 items-start justify-between gap-4 border-b border-line bg-chrome-header px-5 py-4">
      <div class="min-w-0">
        <h2 class="m-0 truncate text-lg font-semibold">{selectedRun.ruleName}</h2>
        <p class="m-0 mt-1 text-xs text-fg-muted">
          {selectedRun.targetKind || 'target'} {selectedRun.targetName || 'No target'} &middot; {selectedRun.trigger || 'manual'} &middot; {formatDateTime(selectedRun.startedAt)}
        </p>
      </div>
      <Badge variant={selectedRun.status} class="shrink-0">{String(selectedRun.status || 'unknown').toUpperCase()}</Badge>
    </header>

    <div class="flex-1 min-h-0 overflow-auto p-5">
      {#if selectedRun.error}
        <div class="mb-4 rounded border border-danger-500/40 bg-danger-500/10 px-3 py-2 text-sm text-danger-200">
          {selectedRun.error}
        </div>
      {/if}

      <pre class="m-0 min-h-full overflow-auto rounded border border-line bg-panel p-4 font-mono text-sm leading-tight-cli text-fg whitespace-pre-wrap">{output}</pre>
    </div>
  {:else}
    <EmptyState icon="clipboard-list" title="No run selected" description="Select a run from the left to inspect its output." class="m-4" />
  {/if}
</div>
