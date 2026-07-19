<script>
  import ErrorState from '../ui/ErrorState.svelte'
  import EmptyState from '../ui/EmptyState.svelte'

  let {
    error = null,
    empty = false,
    emptyIcon = 'info',
    emptyTitle = 'No items',
    emptyDescription = '',
    class: className = '',
    children,
  } = $props()
</script>

<div class="flex-1 overflow-auto min-h-0 {className}">
  {#if error && empty}
    <ErrorState {error} />
  {:else if empty}
    <EmptyState icon={emptyIcon} title={emptyTitle} description={emptyDescription} />
  {:else if children}
    {@render children()}
  {/if}
</div>
