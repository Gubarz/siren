<script>
  import Alert from './Alert.svelte'

  let {
    title = 'Something went wrong',
    error: errorMsg = '',
    class: className = '',
    children,
  } = $props()

  const message = $derived(
    errorMsg instanceof Error ? errorMsg.message : String(errorMsg)
  )
</script>

<div class={className}>
  <Alert color="red">
    <p class="font-medium">{title}</p>
    {#if message}
      <p class="text-sm mt-1 opacity-80">{message}</p>
    {/if}
    {#if children}
      <div class="mt-2">
        {@render children()}
      </div>
    {/if}
  </Alert>
</div>
