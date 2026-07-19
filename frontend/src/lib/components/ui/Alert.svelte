<script>
  import { Alert as FlowbiteAlert } from 'flowbite-svelte'
  import Icon from './Icon.svelte'

  let {
    color = 'primary',
    icon: iconName = 'info',
    dismissable = false,
    class: className = '',
    children,
  } = $props()
</script>

<FlowbiteAlert {color} class={className} {dismissable}>
  {#snippet icon()}
    <Icon name={iconName} size={20} />
  {/snippet}
  {#if children}
    {@render children()}
  {/if}
</FlowbiteAlert>
