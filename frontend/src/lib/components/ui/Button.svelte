<script>
  import { Button as FlowbiteButton } from 'flowbite-svelte'
  import Icon from './Icon.svelte'

  let {
    color = undefined,
    size = 'md',
    icon = '',
    loading = false,
    disabled = false,
    type = 'button',
    href = '',
    fullWidth = false,
    class: className = '',
    onclick,
    children,
    ...restProps
  } = $props()

  const tag = $derived(href ? 'a' : 'button')
  const isDisabled = $derived(disabled || loading)
  const buttonType = $derived(href ? undefined : type)
  const baseClass = 'shrink-0 gap-1 whitespace-nowrap'
  const buttonClass = $derived([
    baseClass,
    fullWidth ? 'w-full' : '',
    className,
  ].filter(Boolean).join(' '))
</script>

<FlowbiteButton
  {size}
  {color}
  {tag}
  disabled={isDisabled}
  {loading}
  href={href || undefined}
  type={buttonType}
  class={buttonClass}
  {onclick}
  {...restProps}
>
  {#if icon}
    <Icon name={icon} size={12} class="shrink-0" />
  {/if}
  {#if children}
    {@render children()}
  {/if}
</FlowbiteButton>
