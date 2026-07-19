<script>
  import Button from './Button.svelte'

  let {
    icon,
    label,
    color = undefined,
    size = 'sm',
    loading = false,
    disabled = false,
    tooltip = '',
    title = '',
    onclick,
    ...restProps
  } = $props()

  let buttonTitle = $derived(tooltip || title || label)
</script>

<Button
  {color}
  {size}
  {icon}
  {loading}
  {disabled}
  aria-label={label}
  title={buttonTitle}
  {onclick}
  {...restProps}
/>
