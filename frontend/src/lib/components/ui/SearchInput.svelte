<script>
  import Icon from './Icon.svelte'

  let {
    value = $bindable(''),
    placeholder = 'Search…',
    class: className = '',
    oninput,
    onchange,
    ...restProps
  } = $props()
</script>

<div class="relative {className}">
  <Icon name="search" size={14} class="absolute left-3 top-1/2 -translate-y-1/2 text-fg-muted" />
  <input
    type="text"
    bind:value
    {placeholder}
    class="w-full bg-surface-100 text-fg border border-line rounded px-8 py-1 text-sm outline-none focus:border-primary-500"
    {oninput}
    {onchange}
    {...restProps}
  />
  {#if value}
    <button
      type="button"
      class="absolute right-2 top-1/2 -translate-y-1/2 text-fg-muted hover:text-fg"
      onclick={() => value = ''}
      aria-label="Clear search"
    >
      <Icon name="x" size={14} />
    </button>
  {/if}
</div>
