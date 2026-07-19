<script>
  import Select from '$components/ui/Select.svelte'

  let {
    value = $bindable(''),
    label = '',
    description = '',
    options = [],
    placeholder = '',
    id = '',
    disabled = false,
    size = 'md',
    menuClass = '',
    class: className = '',
    onchange,
    ...restProps
  } = $props()

  let fieldId = $derived(id || `sel-${String(label || 'field').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')}`)
</script>

<div class="mb-2">
  <label class="block text-base font-medium text-fg mb-1" for={fieldId}>{label}</label>
  <Select
    id={fieldId}
    bind:value
    {options}
    {placeholder}
    {disabled}
    {size}
    {menuClass}
    class={className}
    {onchange}
    {...restProps}
  />
  {#if description}
    <span class="block text-xs mt-1 text-fg-muted">{description}</span>
  {/if}
</div>
