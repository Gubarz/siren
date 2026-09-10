<script module>
  // Single source of truth for the variant → class mapping; exported so
  // tests pin the mapping rather than a class the fallback also yields.
  export const STATUS_DOT_VARIANTS = {
    online: 'bg-success-500',
    offline: 'bg-danger-500',
    pending: 'bg-warning-500',
    warning: 'bg-warning-500',
    discovered: 'bg-device-500',
    lost: 'bg-gray-500',
  }
</script>

<script>
  let { variant = 'offline', size = 8, glow = true, label = '' } = $props()

  const color = $derived(STATUS_DOT_VARIANTS[variant] || 'bg-gray-500')

  const shadow = $derived(glow ? 'status-dot-glow shadow-current' : '')
</script>

<span
  class="inline-block rounded-full {color} {shadow}"
  style="width: {size}px; height: {size}px;"
  role="status"
  aria-label={label || variant}
></span>
