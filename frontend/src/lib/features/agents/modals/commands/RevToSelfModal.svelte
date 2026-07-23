<script>
  import Modal from '$components/patterns/Modal.svelte'
  import Button from '$components/ui/Button.svelte'

  let {
    open = $bindable(false),
    onexecute,
    onclose,
  } = $props()

  function execute() {
    onexecute?.({ cmd: 'rev2self' })
  }
</script>

<Modal bind:open title="Rev2Self" size="md" {onclose}>
  <p class="text-fg-muted text-sm mb-4">
    Revert to the implant's original process token. This drops any impersonation
    from <code>getsystem</code>, <code>make-token</code>, or <code>runas</code>.
  </p>

  {#snippet footer()}
    <div class="flex justify-end gap-2">
      <Button color="dark" onclick={() => open = false}>Cancel</Button>
      <Button color="warning" onclick={execute}>Rev2Self</Button>
    </div>
  {/snippet}
</Modal>
