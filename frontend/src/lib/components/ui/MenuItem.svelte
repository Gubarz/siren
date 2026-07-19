<script>
  import { DropdownItem } from 'flowbite-svelte'

  let {
    disabled = false,
    variant = 'default',
    class: className = '',
    children,
    ...restProps
  } = $props()

  const baseClass = '!flex !w-full items-center gap-2 !px-2 !py-1 !text-left !text-sm hover:!bg-row-hover'
  let variantClass = $derived({
    default: '!text-fg',
    accent: '!text-brand',
    danger: '!text-danger-500',
  }[variant] ?? '!text-fg')
</script>

<DropdownItem
  {disabled}
  class="{baseClass} {variantClass} {className}"
  {...restProps} 
>
  {@render children()}
</DropdownItem>