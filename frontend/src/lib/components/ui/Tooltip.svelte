<script>
  let {
    text,
    class: className = '',
    children,
  } = $props()
</script>

<span class={`inline-flex ${className}`} title={text}>
  {@render children()}
</span>
