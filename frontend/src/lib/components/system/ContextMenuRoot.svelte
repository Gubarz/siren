<script>
  import { contextMenu } from '../../stores/ui/contextMenu.svelte.js'
  import Icon from '../ui/Icon.svelte'
  import Divider from '../ui/Divider.svelte'

  let menuState = $derived(contextMenu)

  let menuEl = $state(null)
  let menuX = $state(0)
  let menuY = $state(0)
  let activeSubmenu = $state(null)
  let submenuStyle = $state('')
  let submenuTriggerRects = $state({})
  let menuToken = $state('')
  let lastActivatedAt = 0

  function documentZoom() {
    const inlineZoom = Number(document.documentElement.style.zoom)
    if (Number.isFinite(inlineZoom) && inlineZoom > 0) return inlineZoom

    const computedZoom = Number(getComputedStyle(document.documentElement).zoom)
    if (Number.isFinite(computedZoom) && computedZoom > 0) return computedZoom

    return 1
  }

  $effect(() => {
    if (menuState.isOpen && menuEl) {
      const zoom = documentZoom()
      const rect = menuEl.getBoundingClientRect()
      const winW = window.innerWidth
      const winH = window.innerHeight
      let visualX = menuState.x + rect.width > winW ? winW - rect.width - 8 : menuState.x
      let visualY = menuState.y + rect.height > winH ? winH - rect.height - 8 : menuState.y
      if (visualX < 0) visualX = 8
      if (visualY < 0) visualY = 8
      menuX = visualX / zoom
      menuY = visualY / zoom
    }
  })

  $effect(() => {
    const nextToken = menuState.isOpen
      ? `${menuState.x}:${menuState.y}:${menuState.sections.length}:${String(menuState.target?.ID || menuState.target?.key || '')}`
      : ''
    if (nextToken === menuToken) return
    menuToken = nextToken
    activeSubmenu = null
    submenuStyle = ''
    submenuTriggerRects = {}
  })

  function activateItem(event, item) {
    event.stopPropagation()
    if (event.type === 'mousedown') event.preventDefault()
    if (item.children?.length || item.disabled) return

    const now = performance.now()
    if (now - lastActivatedAt < 80) return
    lastActivatedAt = now

    item.on?.()
    // Defer removal so the click event that follows mousedown still fires
    // inside the context menu DOM (where stopPropagation catches it) instead
    // of leaking through to whatever renders underneath (e.g. a modal backdrop).
    requestAnimationFrame(() => contextMenu.close())
  }

  function openSubmenu(name, event) {
    // Capture the trigger rect synchronously (the mouse event carries it);
    // positioning waits for the submenu DOM in the $effect below.
    submenuTriggerRects[name] = event.currentTarget.getBoundingClientRect()
    activeSubmenu = name
  }

  // Re-position the submenu whenever it opens or the trigger rect updates.
  // Post-DOM $effect timing means the submenu is already mounted when we
  // measure it.
  $effect(() => {
    if (activeSubmenu) positionSubmenu(activeSubmenu)
  })

  function positionSubmenu(name) {
    if (!menuEl) return
    const submenuEl = menuEl.querySelector(`[data-submenu="${name}"]`)
    const triggerRect = submenuTriggerRects[name]
    if (!submenuEl || !triggerRect) return
    const subRect = submenuEl.getBoundingClientRect()
    const winW = window.innerWidth
    const winH = window.innerHeight
    let sx = triggerRect.right + 2
    if (sx + subRect.width > winW) sx = triggerRect.left - subRect.width - 2
    if (sx < 0) sx = 4
    let sy = triggerRect.top
    if (sy + subRect.height > winH) sy = winH - subRect.height - 4
    if (sy < 0) sy = 4
    submenuStyle = `top: ${sy}px; left: ${sx}px;`
  }

  const handleKeydown = (e) => { if (e.key === 'Escape') contextMenu.close() }
  const handleGlobalClick = () => contextMenu.close()
  const stopPropagation = (e) => e.stopPropagation()
  const handleMenuKeydown = (e) => {
    e.stopPropagation()
    if (e.key === 'Escape') contextMenu.close()
  }
</script>

<svelte:window onkeydown={handleKeydown} onclick={handleGlobalClick} />

{#if menuState.isOpen}
  <div
    bind:this={menuEl}
    class="fixed z-popover min-w-55 border border-panel-border rounded-lg shadow-lg py-1 overflow-y-auto max-h-vh-90 bg-panel text-fg"
    style="top: {menuY}px; left: {menuX}px;"
    role="menu"
    tabindex="-1"
    onclick={stopPropagation}
    onkeydown={handleMenuKeydown}
  >
    {#each menuState.sections as section}
      {#if section.divider}
        <Divider />
      {:else}
        {#if section.title}
          <div class="px-3 py-1 text-xs font-medium text-fg-muted uppercase tracking-wider">{section.title}</div>
        {/if}
        {#each section.items ?? [section] as item}
          {#if item.label}
            <div class="relative">
              <button
                type="button"
                class="w-full flex items-center gap-2 px-3 py-2 text-sm text-fg hover:bg-brand hover:text-white text-left disabled:opacity-50"
                class:bg-brand={item.children && activeSubmenu === item.label}
                role="menuitem"
                aria-haspopup={item.children ? 'menu' : undefined}
                aria-expanded={item.children ? activeSubmenu === item.label : undefined}
                disabled={item.disabled}
                onmousedown={(e) => activateItem(e, item)}
                onclick={(e) => activateItem(e, item)}
                onmouseenter={(e) => { if (item.children) openSubmenu(item.label, e); else activeSubmenu = null }}
              >
                {#if item.icon}
                  <Icon name={item.icon} size={14} class="shrink-0 {item.danger ? 'text-danger-500' : 'text-fg-muted'}" />
                {/if}
                <span class:!text-danger-500={item.danger}>{item.label}</span>
                {#if item.shortcut}
                  <span class="ml-auto text-xs text-fg-muted">{item.shortcut}</span>
                {:else if item.children}
                  <Icon name="chevron-right" size={12} class="ml-auto text-fg-muted" />
                {/if}
              </button>
              {#if item.children && activeSubmenu === item.label}
                <div
                  data-submenu={item.label}
                  class="fixed z-popover min-w-55 border border-panel-border rounded-lg shadow-lg py-1 overflow-y-auto max-h-vh-90 bg-panel text-fg"
                  style={submenuStyle}
                  role="menu"
                >
                  {#each item.children as child}
                    {#if child.label}
                      <button
                        type="button"
                        class="w-full flex items-center gap-2 px-3 py-2 text-sm text-fg hover:bg-brand hover:text-white text-left disabled:opacity-50"
                        role="menuitem"
                        disabled={child.disabled}
                        title={child.description || ''}
                        onmousedown={(e) => activateItem(e, child)}
                        onclick={(e) => activateItem(e, child)}
                      >
                        {#if child.icon}
                          <Icon name={child.icon} size={14} class="shrink-0 text-fg-muted" />
                        {/if}
                        <span class={child.disabled ? 'opacity-60' : ''}>{child.label}</span>
                      </button>
                    {/if}
                  {/each}
                </div>
              {/if}
            </div>
          {/if}
        {/each}
      {/if}
    {/each}
  </div>
{/if}
