<script>
  let {
    label = '',
    hint = '',
    error = '',
    required = false,
    htmlFor = '',
    class: className = '',
    children,
  } = $props()

  let descId = $derived(htmlFor ? `${htmlFor}-desc` : '')
</script>

<div class="flex flex-col gap-1 {className}">
  {#if label}
    <label for={htmlFor} class="text-sm text-fg font-medium">
      {label}
      {#if required}
        <span class="text-danger-500">*</span>
      {/if}
    </label>
  {/if}
  {@render children()}
  {#if error}
    <p id={descId} class="text-xs text-danger-500">{error}</p>
  {:else if hint}
    <p id={descId} class="text-xs text-fg-muted">{hint}</p>
  {/if}
</div>
