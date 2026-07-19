<script>
  import Checkbox from '../ui/Checkbox.svelte'

  let { checked = $bindable(false), label = '', description = '' } = $props()
</script>

<div class="mb-1">
  <Checkbox bind:checked {label} />
  {#if description}
    <span class="block text-xs mt-1 ml-5 text-fg-muted">{description}</span>
  {/if}
</div>
