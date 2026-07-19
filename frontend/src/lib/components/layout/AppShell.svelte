<script>
  let { children } = $props()
</script>

<div class="w-full h-full flex flex-col overflow-hidden bg-canvas text-fg">
  {@render children()}
</div>