<script>
  import TextInput from '../ui/TextInput.svelte'

  let { value = $bindable(''), label = '', description = '', placeholder = '', type = 'text', required = false } = $props()
</script>

<div class="mb-2">
  <label class="block text-base font-medium text-fg mb-1" for="tf-{label}">
    {label}
    {#if required}<span class="text-danger-500 ml-1">*</span>{/if}
  </label>
  <TextInput
    id="tf-{label}"
    {type}
    size="sm"
    bind:value
    {placeholder}
  />
  {#if description}
    <span class="block text-xs mt-1 text-fg-muted">{description}</span>
  {/if}
</div>
