<script>
  import Icon from './Icon.svelte'

  let {
    icon = 'info',
    title = '',
    description = '',
    class: className = '',
    children,
  } = $props()
</script>

<div class="flex flex-col items-center justify-center py-12 px-4 text-center {className}">
  {#if icon}
    <Icon name={icon} size={40} class="text-fg-muted mb-4 opacity-50" />
  {/if}
  {#if title}
    <h3 class="text-lg font-medium text-fg mb-1">{title}</h3>
  {/if}
  {#if description}
    <p class="text-sm text-fg-muted max-w-md">{description}</p>
  {/if}
  {#if children}
    <div class="mt-4">
      {@render children()}
    </div>
  {/if}
</div>
