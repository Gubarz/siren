<script>
  let {
    checked = false,
    value = '',
    name = '',
    label = '',
    disabled = false,
    class: className = '',
    onchange,
  } = $props()
</script>

<label class="inline-flex items-center gap-2 cursor-pointer {className}">
  <input
    type="radio"
    {checked}
    {value}
    {name}
    {disabled}
    class="text-primary-500 focus:ring-primary-500"
    {onchange}
  />
  {#if label}
    <span class="text-sm text-fg">{label}</span>
  {/if}
</label>
