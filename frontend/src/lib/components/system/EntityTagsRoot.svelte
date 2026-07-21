<script>
  import EntityTagsModal from '../../features/tags/EntityTagsModal.svelte'
  import { tagsModal } from '../../stores/ui/tagsModal.svelte.js'
</script>

<EntityTagsModal
  bind:open={tagsModal.open}
  entityType={tagsModal.entityType}
  entityID={tagsModal.entityID}
  entityLabel={tagsModal.entityLabel}
  entities={tagsModal.entities}
/>
