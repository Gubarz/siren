<script>
  import EntityCommentsModal from '../../features/comments/EntityCommentsModal.svelte'
  import { commentsModal } from '../../stores/ui/commentsModal.svelte.js'
</script>

<EntityCommentsModal
  bind:open={commentsModal.open}
  entityType={commentsModal.entityType}
  entityID={commentsModal.entityID}
  entityLabel={commentsModal.entityLabel}
/>
