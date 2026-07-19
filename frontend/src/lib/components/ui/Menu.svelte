<script>
  import { Dropdown } from 'flowbite-svelte'

  let {
    isOpen = $bindable(false),
    placement = 'bottom-start',
    offset = 4,
    minWidth = '9rem',
    class: className = '',
    children,
    ...restProps
  } = $props()

  const baseClass = 'z-popover !mt-0 overflow-hidden rounded border border-line !bg-panel !text-fg !shadow-panel'
</script>

<Dropdown
  simple
  bind:isOpen
  {placement}
  {offset}
  triggerDelay={0}
  transitionParams={{ duration: 0 }}
  class={`${baseClass} ${className}`}
  style={`min-width: ${minWidth}`}
  {...restProps}
>
  {@render children()}
</Dropdown>
