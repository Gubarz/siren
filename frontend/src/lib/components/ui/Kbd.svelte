<script>
  import { Kbd as FlowbiteKbd } from 'flowbite-svelte'

  let { key, class: className = '' } = $props()
</script>

<FlowbiteKbd class={className}>{key}</FlowbiteKbd>
