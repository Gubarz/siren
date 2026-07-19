<script>
  import Icon from './Icon.svelte'

  let {
    description = 'Loading...',
    class: className = '',
  } = $props()
</script>

<div class="flex flex-col items-center justify-center py-12 px-4 text-center {className}">
  <Icon name="loader" size={40} class="text-brand mb-4 animate-spin" />
  {#if description}
    <p class="text-sm text-fg-muted">{description}</p>
  {/if}
</div>
