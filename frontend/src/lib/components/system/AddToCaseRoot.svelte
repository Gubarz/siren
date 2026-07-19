<script>
  import AddToCaseModal from '../../features/server/cases/AddToCaseModal.svelte'
  import { addToCase } from '../../stores/ui/addToCase.svelte.js'

  // Single root instance so any feature panel can trigger the modal via
  // addToCase.open({collection, itemID, label}). Mounted next to
  // GlobalDialog / Toasts / ContextMenuRoot in App.svelte.

</script>

<AddToCaseModal
  open={addToCase.isOpen}
  collection={addToCase.collection}
  itemID={addToCase.itemID}
  label={addToCase.label}
  onclose={() => addToCase.close()}
/>
