<script>
  import Modal from '../../../../components/patterns/Modal.svelte'
  import CommandPreview from './CommandPreview.svelte'
  import CommandModalFooter from './CommandModalFooter.svelte'

  let {
    open = $bindable(false),
    title = '',
    size = '2xl',
    onclose,
    initialValues = {},
    onreset,
    cmdPreview = '',
    commandPath = '',
    currentValues = {},
    onapply,
    primaryLabel = 'Execute',
    onprimary,
    primaryDisabled = false,
    children,
  } = $props()

  // Saved initial values are applied before paint so the fields are populated
  // on the first render of the modal.
  $effect.pre(() => onreset?.(initialValues))
</script>

<Modal bind:open {title} {size} {onclose}>
  {@render children?.()}

  <CommandPreview cmd={cmdPreview} />

  {#snippet footer()}
    <CommandModalFooter
      {commandPath}
      {currentValues}
      {onapply}
      {primaryLabel}
      onprimary={onprimary}
      {primaryDisabled}
      oncancel={() => open = false}
    />
  {/snippet}
</Modal>
