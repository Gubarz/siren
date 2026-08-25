<script>
  import Badge from '$components/ui/Badge.svelte';
  import Button from '$components/ui/Button.svelte';
  import BhGraph from './BhGraph.svelte';

  let {
    title = '',
    entities = [],
    graph = null,
    loading = false,
    error = '',
    showGraph = $bindable(false),
    actionsFor = () => [],
    onEdgeClick = null,
  } = $props();

  let expandedRow = $state('');
</script>

<div>
  <div class="flex items-center gap-2 mb-2">
    <h4 class="m-0 text-fg text-sm">{title}</h4>
    <span class="text-xs text-fg-muted">{entities.length}</span>
    <span class="flex-1"></span>
    <Button size="xs" color="alternative" onclick={() => (showGraph = !showGraph)}>
      {showGraph ? 'List view' : 'Graph view'}
    </Button>
  </div>
  {#if loading}
    <p class="text-xs text-fg-muted m-0">Loading…</p>
  {:else if error}
    <p class="text-xs text-danger-500 m-0">{error}</p>
  {:else if showGraph && graph}
    <BhGraph {graph} {onEdgeClick} />
  {:else if entities.length === 0}
    <p class="text-xs text-fg-muted m-0">None found.</p>
  {:else}
    <ul class="m-0 p-0 list-none flex flex-col">
      {#each entities as entity (entity.id)}
        {@const actions = actionsFor(entity)}
        <li class="border-b border-line last:border-b-0 py-1">
          <!-- eslint-disable-next-line local/no-raw-button -->
          <button
            type="button"
            class="w-full flex items-center gap-2 bg-transparent border-0 p-0 text-left cursor-pointer hover:bg-row-hover"
            aria-expanded={expandedRow === entity.id}
            onclick={() => (expandedRow = expandedRow === entity.id ? '' : entity.id)}
          >
            <span class="text-xs text-fg truncate flex-1" title={entity.label}>{entity.label}</span>
            {#if entity.kind}
              <Badge>{entity.kind}</Badge>
            {/if}
            {#if entity.tierZero}
              <Badge variant="danger">Tier 0</Badge>
            {/if}
            {#if entity.owned}
              <Badge variant="success">Owned</Badge>
            {/if}
          </button>
          {#if expandedRow === entity.id && actions.length > 0}
            <div class="flex flex-wrap items-center gap-2 pt-1">
              {#each actions as action (action.label)}
                <Button
                  size="xs"
                  color={action.disabled ? 'alternative' : 'primary'}
                  disabled={action.disabled}
                  title={action.reason || ''}
                  onclick={action.on}
                >
                  {action.label}
                </Button>
              {/each}
            </div>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}
</div>
