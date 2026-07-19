<script>
  import { Textarea as FlowbiteTextarea } from 'flowbite-svelte'

  let {
    value = $bindable(''),
    placeholder = '',
    disabled = false,
    readonly = false,
    autofocus = false,
    rows = undefined,
    class: className = '',
    divClass = '',
    onchange,
    oninput,
    onkeydown,
    element = $bindable(null),
    ...restProps
  } = $props()
</script>

<FlowbiteTextarea
  bind:value
  {placeholder}
  {disabled}
  {readonly}
  {autofocus}
  {rows}
  class="bg-chrome! border-line! text-fg! placeholder:text-fg-muted! focus:border-brand! focus:ring-brand/40! {className}"
  {divClass}
  {onchange}
  {oninput}
  {onkeydown}
  bind:elementRef={element}
  {...restProps}
/>
