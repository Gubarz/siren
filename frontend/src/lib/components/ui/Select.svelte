<script>
  import { Dropdown, DropdownItem } from 'flowbite-svelte'
  import Icon from './Icon.svelte'

  let {
    value = $bindable(''),
    options = [],
    placeholder = 'Choose option ...',
    disabled = false,
    size = 'md',
    menuClass = '',
    class: className = '',
    onchange,
    ...restProps
  } = $props()

  let isOpen = $state(false)

  const sizeClasses = {
    sm: 'h-8 px-2 py-1 text-sm',
    md: 'h-8 px-2 py-1 text-sm',
    lg: 'h-9 px-3 py-2 text-base',
  }
  const selectMenuClass = 'z-popover !mt-0 w-(--trigger-width) min-w-28 overflow-hidden rounded border border-line !bg-panel !text-fg !shadow-panel !divide-line'
  const selectItemBaseClass = '!flex !w-full items-center justify-between gap-2 !px-2 !py-1 !text-left !text-sm !text-fg hover:!bg-row-hover hover:!text-fg focus:!bg-row-hover focus:!text-fg disabled:pointer-events-none disabled:opacity-50'
  const selectItemActiveClass = '!bg-brand !text-on-brand hover:!bg-brand hover:!text-on-brand focus:!bg-brand focus:!text-on-brand'
  const selectTransitionParams = { duration: 0 }

  let normalizedOptions = $derived(options.map((option) => {
    if (typeof option === 'string') {
      return { value: option, label: option, disabled: false }
    }
    return {
      value: option.value,
      label: option.label ?? option.name ?? option.value,
      disabled: Boolean(option.disabled),
    }
  }))
  let selectedOption = $derived(normalizedOptions.find((option) => option.value === value))
  let displayValue = $derived(selectedOption?.label ?? placeholder)
  let triggerClass = $derived([
    'inline-flex w-full items-center justify-between gap-2 rounded border border-line bg-canvas text-left text-fg outline-none hover:border-brand focus:border-brand',
    sizeClasses[size] ?? sizeClasses.md,
    disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',
    className,
  ].filter(Boolean).join(' '))
  let dropdownClass = $derived([selectMenuClass, menuClass].filter(Boolean).join(' '))

  function optionClass(option) {
    return option.value === value ? `${selectItemBaseClass} ${selectItemActiveClass}` : selectItemBaseClass
  }

  function selectOption(option) {
    if (option.disabled) return
    value = option.value
    isOpen = false
    onchange?.(value, option)
  }

  function syncDropdownWidth(event) {
    const width = event?.trigger?.getBoundingClientRect?.().width
    event?.currentTarget?.style?.setProperty('--trigger-width', `${width || 112}px`)
  }
</script>

<button
  type="button"
  disabled={disabled}
  aria-haspopup="listbox"
  aria-expanded={isOpen}
  class={triggerClass}
  {...restProps}
>
  <span class="truncate" class:text-fg-muted={!selectedOption}>{displayValue}</span>
  <Icon name="chevron-down" size={14} class="shrink-0 text-fg-muted transition-transform {isOpen ? 'rotate-180' : ''}" />
</button>
<Dropdown
  simple
  bind:isOpen
  placement="bottom-start"
  offset={0}
  triggerDelay={0}
  transitionParams={selectTransitionParams}
  role="listbox"
  class={dropdownClass}
  onbeforetoggle={syncDropdownWidth}
>
  {#each normalizedOptions as option}
    <DropdownItem
      onclick={() => selectOption(option)}
      disabled={option.disabled}
      class={optionClass(option)}
      role="option"
      aria-selected={option.value === value}
    >
      <span class="truncate">{option.label}</span>
    </DropdownItem>
  {/each}
</Dropdown>
