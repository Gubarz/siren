<script>
  let {
    cmdPreview = '',
    error = '',
    resultPath = '',
  } = $props()
</script>

<div class="mb-4">
  <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
  <code class="block break-all whitespace-pre-wrap rounded border border-line bg-chrome p-2 font-mono text-xs text-fg">{cmdPreview}</code>
</div>

{#if error}<div class="mt-2 whitespace-pre-wrap text-sm text-danger-500">{error}</div>{/if}
{#if resultPath}<div class="mt-2 break-all text-sm text-success-500">{resultPath}</div>{/if}
