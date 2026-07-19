<script>
  import { icons } from './icons.js'

  let { name, size = 16, class: className = '' } = $props()
  const IconComponent = $derived(icons[name])
</script>

{#if IconComponent}
  <IconComponent {size} class={className} />
{/if}
