<script>
  let {
    checked = $bindable(false),
    label = '',
    disabled = false,
    class: className = '',
    onchange,
    ...restProps
  } = $props()
</script>

<label class="inline-flex items-center gap-2 cursor-pointer {className}">
  <input
    type="checkbox"
    bind:checked
    {disabled}
    class="rounded border-line bg-surface-100 text-primary-500 focus:ring-primary-500"
    {onchange}
    {...restProps}
  />
  {#if label}
    <span class="text-sm text-fg">{label}</span>
  {/if}
</label>
