<script>
  let { cmd = '' } = $props()
</script>

<div class="mb-4">
  <span class="block text-sm font-semibold text-fg mb-1">Command preview</span>
  <code class="block p-2 border border-line rounded bg-chrome text-fg break-all">{cmd}</code>
</div>
