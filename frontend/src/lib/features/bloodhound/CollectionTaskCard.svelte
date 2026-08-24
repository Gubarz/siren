<script>
  import Badge from '$components/ui/Badge.svelte';

  let { collection = null } = $props();

  const stageVariant = {
    done: 'success', failed: 'danger', staged: 'default', running: 'primary',
    collecting: 'primary', downloading: 'primary', ingesting: 'primary',
  };

  let variant = $derived(stageVariant[collection?.stage] || 'default');
</script>

<div class="flex items-center gap-2 px-4 py-2 border-b border-table-line">
  <span class="text-xs font-mono text-fg-muted">{collection?.id?.slice(0, 8)}</span>
  <Badge variant={variant}>{collection?.stage || 'unknown'}</Badge>
  <span class="flex-1 min-w-0 text-xs text-fg-muted truncate" title={collection?.progress || ''}>
    {collection?.progress || ''}
  </span>
  {#if collection?.error}
    <span class="text-xs text-danger-500 truncate max-w-64" title={collection.error}>{collection.error}</span>
  {/if}
  {#if collection?.ingestJobId}
    <span class="text-xs text-fg-muted shrink-0">job #{collection.ingestJobId}</span>
  {/if}
</div>
