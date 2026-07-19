<script>
  import Icon from '../ui/Icon.svelte'

  let {
    open = $bindable(false),
    title = '',
    icon = '',
    size = 'md',
    position = 'center',
    dismissable = true,
    class: className = '',
    zIndex = null,
    onclose,
    children,
    footer,
  } = $props()

  let panelEl = $state(null)
  let previousFocus = null
  let initialTopPx = $state(null)

  // $effect runs after DOM update by default, so we can focus/scroll
  // directly — no tick().then() wrapper required.
  $effect(() => {
    if (open) {
      previousFocus = document.activeElement
      initialTopPx = null
      if (position === 'top' && panelEl) {
        initialTopPx = Math.max(0, Math.round((window.innerHeight - panelEl.offsetHeight) / 2))
      }
      const target = panelEl?.querySelector('[autofocus], input, textarea, button')
      target?.focus?.()
    } else {
      if (previousFocus && document.contains(previousFocus)) {
        previousFocus.focus()
      }
      previousFocus = null
    }
  })

  function close() {
    if (!dismissable) return
    open = false
    onclose?.()
  }

  let pointerDownOnBackdrop = false
  function handleBackdropPointerDown(e) {
    // Only treat this as a "click on the backdrop" if the drag started on the
    // backdrop itself. Otherwise a text selection that ends outside the panel
    // would dismiss the modal.
    pointerDownOnBackdrop = e.target === e.currentTarget
  }
  function handleBackdropClick(e) {
    if (e.target === e.currentTarget && pointerDownOnBackdrop) close()
    pointerDownOnBackdrop = false
  }

  function handleWindowKeydown(e) {
    if (e.key === 'Escape' && open && dismissable) {
      e.preventDefault()
      close()
    }
  }

  const widthMap = {
    sm: 'max-w-sm',
    md: 'max-w-md',
    lg: 'max-w-lg',
    xl: 'max-w-xl',
    '2xl': 'max-w-2xl',
    '3xl': 'max-w-3xl',
    '4xl': 'max-w-4xl',
    '5xl': 'max-w-5xl',
    full: 'max-w-full mx-4',
  }

  // While measuring (initialTopPx === null), stay center-aligned so the
  // initial render is already at visual center. Once we've measured the
  // panel's height on open, switch to items-start + a padding-top equal to
  // where items-center just put it — pixel-identical position, but now the
  // top edge is pinned so growth extends downward instead of shifting up.
  const positionClass = $derived(
    position === 'top' && initialTopPx !== null ? 'items-start' : 'items-center',
  )

  const backdropStyle = $derived([
    position === 'top' && initialTopPx !== null ? `padding-top: ${initialTopPx}px;` : '',
    zIndex != null ? `z-index: ${zIndex};` : '',
  ].filter(Boolean).join(' '))
</script>

<svelte:window onkeydown={handleWindowKeydown} />

{#if open}
  <div
    class="fixed inset-0 z-100 flex justify-center bg-black/60 backdrop-blur-sm p-4 {positionClass}"
    style={backdropStyle}
    role="presentation"
    onpointerdown={handleBackdropPointerDown}
    onclick={handleBackdropClick}
  >
    <div
      bind:this={panelEl}
      class="rounded-lg border border-line bg-surface-100 text-fg shadow-panel w-full {widthMap[size] || 'max-w-md'} {className}"
      role="dialog"
      aria-modal="true"
      aria-label={title}
    >
      <div class="flex flex-col max-h-vh-80">
        {#if title || icon}
          <div class="flex items-center justify-between px-5 py-3 border-b border-line shrink-0">
            <div class="flex items-center gap-2">
              {#if icon}
                <Icon name={icon} size={18} class="text-brand" />
              {/if}
              <h2 class="text-base font-medium">{title}</h2>
            </div>
            {#if dismissable}
              <button
                type="button"
                class="text-fg-muted hover:text-fg cursor-pointer bg-transparent border-0 p-0"
                onclick={close}
                aria-label="Close"
              >
                <Icon name="x" size={16} />
              </button>
            {/if}
          </div>
        {/if}

        <div class="flex-1 overflow-auto px-5 py-4 min-h-0">
          {#if children}
            {@render children()}
          {/if}
        </div>

        {#if footer}
          <div class="px-5 py-3 border-t border-line bg-surface-50 shrink-0">
            {@render footer()}
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
