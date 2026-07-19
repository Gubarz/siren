<script>
  let { variant = 'offline', size = 8, glow = true, label = '' } = $props()

  const color = $derived({
    online: 'bg-success-500',
    offline: 'bg-danger-500',
    pending: 'bg-warning-500',
    warning: 'bg-warning-500',
    discovered: 'bg-device-500',
  }[variant] || 'bg-gray-500')

  const shadow = $derived(glow ? 'status-dot-glow shadow-current' : '')
</script>

<span
  class="inline-block rounded-full {color} {shadow}"
  style="width: {size}px; height: {size}px;"
  role="status"
  aria-label={label || variant}
></span>
