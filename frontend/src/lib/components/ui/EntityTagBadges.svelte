<script>
  import TagBadge from './TagBadge.svelte'
  import FittedTagBadges from './FittedTagBadges.svelte'
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
  {#if compact}
    <FittedTagBadges {tags} class={className} />
  {:else}
    <div class="flex flex-wrap items-center gap-1 {className}">
      {#each tags as tag}
        <TagBadge {tag} />
      {/each}
    </div>
  {/if}
{:else if showEmpty}
  <span class="text-fg-muted text-xs {className}">-</span>
{/if}
