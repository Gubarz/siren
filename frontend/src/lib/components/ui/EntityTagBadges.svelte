<script>
  import TagBadge from './TagBadge.svelte'
  import { entityTags } from '../../stores/resources/entityTags.svelte.js'
  import { useResource } from '../../stores/lib/createResource.svelte.js'

  useResource(entityTags)

  let {
    entityType = '',
    entityID = '',
    compact = false,
    showEmpty = false,
    class: className = '',
  } = $props()

  let key = $derived(`${String(entityType || '').trim().toLowerCase()}:${String(entityID || '').trim()}`)
  let tags = $derived(entityTags.data?.[key] || [])
</script>

{#if tags.length > 0}
  <div class="flex flex-wrap items-center gap-1 {className}">
    {#each compact ? tags.slice(0, 2) : tags as tag}
      <TagBadge {tag} />
    {/each}
    {#if compact && tags.length > 2}
      <span class="text-3xs text-fg-muted">+{tags.length - 2}</span>
    {/if}
  </div>
{:else if showEmpty}
  <span class="text-fg-muted text-xs {className}">-</span>
{/if}
