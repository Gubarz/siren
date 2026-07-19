<script>
  import Button from '$components/ui/Button.svelte'
  import IconButton from '$components/ui/IconButton.svelte'

  // Compact bar that appears above the SessionsTable when more than one
  // agent is selected. Every button routes to a bulk handler and clears
  // the selection on success so operators don't accidentally re-fire.

  let {
    count = 0,
    onkill = () => {},
    onrename = () => {},
    onaddtag = () => {},
    onremovetag = () => {},
    onclear = () => {},
  } = $props()
</script>

{#if count > 1}
  <div class="">
    <div class="ml-auto flex items-center gap-1">
      <Button color="dark" size="xs" icon="tag" onclick={onaddtag}>Add tag</Button>
      <Button color="dark" size="xs" icon="tag" onclick={onremovetag}>Remove tag</Button>
      <Button color="dark" size="xs" icon="pen" onclick={onrename}>Rename prefix…</Button>
      <Button color="red" size="xs" icon="skull" onclick={onkill}>Kill</Button>
      <IconButton icon="x" label="Clear selection" tooltip="Clear selection" size="xs" onclick={onclear} />
    </div>
  </div>
{/if}
