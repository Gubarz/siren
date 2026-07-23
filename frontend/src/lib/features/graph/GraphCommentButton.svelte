<script>
  import IconButton from '$components/ui/IconButton.svelte'
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js'

  let {
    entityType,
    entityID,
    entityLabel,
    hasComments = false,
  } = $props()

  let label = $derived(hasComments ? 'View comments' : 'Add comment')
  let buttonClass = $derived(
    `nodrag nopan ml-auto p-1! bg-transparent! dark:bg-transparent! ` +
    `hover:bg-row-hover! dark:hover:bg-row-hover! transition-colors ` +
    (hasComments
      ? 'text-brand! hover:text-brand!'
      : 'text-fg-muted! hover:text-fg!'),
  )

  function openComments(event) {
    event.stopPropagation()
    commentsModal.openComments(entityType, entityID, entityLabel)
  }
</script>

<IconButton
  icon="message-square"
  {label}
  size="xs"
  color="dark"
  class={buttonClass}
  onclick={openComments}
/>
