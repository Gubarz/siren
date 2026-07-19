<script>
  let { compact = false, class: className = '', children } = $props()
</script>

<div class="flex items-center gap-2 px-3 py-{compact ? '1' : '2'} border-b border-line bg-surface-50 shrink-0 {className}">
  {@render children()}
</div>
