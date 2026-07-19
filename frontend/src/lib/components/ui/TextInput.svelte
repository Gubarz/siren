<script>
  import { Input } from 'flowbite-svelte'

  let {
    value = $bindable(''),
    placeholder = '',
    disabled = false,
    readonly = false,
    autofocus = false,
    type = 'text',
    size = 'md',
    class: className = '',
    onchange,
    oninput,
    onkeydown,
    element = $bindable(null),
    ...restProps
  } = $props()
</script>

<Input
  {type}
  {size}
  bind:value
  {placeholder}
  {disabled}
  {readonly}
  {autofocus}
  class="bg-chrome! border-line! text-fg! placeholder:text-fg-muted! focus:border-brand! focus:ring-brand/40! {className}"
  {onchange}
  {oninput}
  {onkeydown}
  bind:elementRef={element}
  {...restProps}
/>
