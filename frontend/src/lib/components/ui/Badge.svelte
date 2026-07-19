<script>
  import Icon from './Icon.svelte'

  let {
    variant = 'default',
    size = 'sm',
    icon = '',
    class: className = '',
    children,
  } = $props()

  const tinted = {
    session: 'success', beacon: 'beacon', device: 'device', discovered: 'device',
    running: 'warning', completed: 'success', failed: 'danger',
  }
  const colorClass = $derived({
    default: 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-300',
    primary: 'bg-primary-100 text-primary-800 dark:bg-primary-900 dark:text-primary-300',
    success: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300',
    warning: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300',
    danger: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300',
  }[variant] || '')
  const colorStyle = $derived(tinted[variant]
    ? `color: var(--color-${tinted[variant]}-500); background-color: color-mix(in srgb, var(--color-${tinted[variant]}-500) 20%, transparent);`
    : '')

  const sizeClass = $derived(
    size === 'graph' ? 'text-3xs leading-none font-sans px-3 py-1 rounded-full uppercase tracking-wider font-semibold' :
    size === 'xs' ? 'text-xs px-2 py-1' :
    'text-xs px-2 py-1'
  )

  const iconSize = $derived(size === 'graph' ? 8 : size === 'xs' ? 10 : 12)
</script>

<span
  class="font-medium inline-flex items-center justify-center {sizeClass} {colorClass} {className}"
  style={colorStyle}
  aria-label={!children && icon ? `${variant} badge` : undefined}
>
  {#if icon}
    <Icon name={icon} size={iconSize} class={children ? 'mr-1' : ''} />
  {/if}
  {#if children}
    {@render children()}
  {/if}
</span>
