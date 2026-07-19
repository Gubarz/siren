<script>
  import Icon from '../ui/Icon.svelte'
  import Badge from '../ui/Badge.svelte'
  import Menu from '../ui/Menu.svelte'
  import MenuItem from '../ui/MenuItem.svelte'

  let {
    tabs = [],
    active = $bindable(''),
    variant = 'underline',
    fullWidth = false,
    maxVisible = 0,
    class: className = '',
    onchange,
    children,
  } = $props()

  let tabListEl = $state(null)
  const base = 'px-3 py-2 text-sm font-medium whitespace-nowrap border-b-2 transition-colors cursor-pointer'
  const variantClass = $derived({
    underline: '',
    pills: 'border-0 rounded-lg',
    boxed: 'border-0 rounded-t-lg border border-line border-b-0',
  }[variant] || '')
  const overflowEnabled = $derived(maxVisible > 0 && tabs.length > maxVisible)
  const visibleTabs = $derived(overflowEnabled ? tabsForRail(tabs, active, maxVisible) : tabs)
  const overflowTabs = $derived(overflowEnabled ? tabs.filter((tab) => !visibleTabs.some((visible) => visible.id === tab.id)) : [])
  let overflowOpen = $state(false)

  function tabsForRail(allTabs, activeId, limit) {
    const visibleCount = Math.max(1, Math.floor(limit))
    const firstTabs = allTabs.slice(0, visibleCount)
    if (firstTabs.some((tab) => tab.id === activeId)) return firstTabs

    const activeTab = allTabs.find((tab) => tab.id === activeId)
    if (!activeTab) return firstTabs

    return [...allTabs.slice(0, Math.max(0, visibleCount - 1)), activeTab]
  }

  function selectTab(tab) {
    overflowOpen = false
    if (!onchange) active = tab.id
    onchange?.(tab.id)
  }

  function handleContextMenu(event, tab) {
    event.preventDefault()
    tab.oncontextmenu?.(event)
  }

  function closeTab(event, tab) {
    event.stopPropagation()
    tab.onclose?.()
  }

  function handleCloseKeydown(event, tab) {
    if (event.key !== 'Enter' && event.key !== ' ') return
    event.preventDefault()
    event.stopPropagation()
    tab.onclose?.()
  }

  function handleWheel(event) {
    if (!tabListEl || tabListEl.scrollWidth <= tabListEl.clientWidth) return
    if (Math.abs(event.deltaY) <= Math.abs(event.deltaX)) return
    event.preventDefault()
    tabListEl.scrollLeft += event.deltaY
  }
</script>

<div class={className}>
  <div
    bind:this={tabListEl}
    class="scrollbar-hidden flex min-w-0 gap-1 overflow-x-auto overflow-y-hidden {fullWidth ? 'w-full' : ''} {variant === 'underline' ? 'border-b border-line' : ''}"
    role="tablist"
    tabindex="-1"
    onwheel={handleWheel}
  >
    {#each visibleTabs as tab (tab.id)}
      <button
        type="button"
        data-tab-button="true"
        draggable={!!tab.ondragstart}
        ondragstart={tab.ondragstart}
        ondragend={tab.ondragend}
        class="{base} {variantClass} {fullWidth ? 'flex-1 text-center' : ''} {active === tab.id ? 'text-brand border-brand' : 'text-fg-muted border-transparent hover:text-fg hover:border-fg-muted'}"
        onclick={() => selectTab(tab)}
        oncontextmenu={(e) => handleContextMenu(e, tab)}
      >
        <span class="flex items-center gap-2 justify-center">
          {#if tab.icon}
            <Icon name={tab.icon} size={14} />
          {/if}
          {tab.label}
          {#if tab.badge}
            <Badge variant={tab.badgeVariant || 'default'}>{tab.badge}</Badge>
          {/if}
          {#if tab.onclose}
            <span
              role="button"
              tabindex="0"
              class="ml-1 px-1 rounded text-xs opacity-60 hover:opacity-100 hover:bg-danger-500 hover:text-white"
              onclick={(e) => closeTab(e, tab)}
              onkeydown={(e) => handleCloseKeydown(e, tab)}
            >&times;</span>
          {/if}
        </span>
      </button>
    {/each}

    {#if overflowTabs.length > 0}
      <div class="relative shrink-0">
        <button
          type="button"
          class="{base} {variantClass} text-fg-muted border-transparent hover:text-fg hover:border-fg-muted"
          aria-haspopup="true"
          aria-expanded={overflowOpen}
        >
          <span class="flex items-center gap-2 justify-center">
            More
            <Badge variant="default">{overflowTabs.length}</Badge>
            <Icon name="chevron-down" size={12} />
          </span>
        </button>
        <Menu bind:isOpen={overflowOpen} minWidth="12rem">
          {#each overflowTabs as tab (tab.id)}
            <MenuItem onclick={() => selectTab(tab)}>
              {#if tab.icon}
                <Icon name={tab.icon} size={14} />
              {/if}
              <span class="min-w-0 truncate">{tab.label}</span>
              {#if tab.badge}
                <Badge variant={tab.badgeVariant || 'default'}>{tab.badge}</Badge>
              {/if}
            </MenuItem>
          {/each}
        </Menu>
      </div>
    {/if}
  </div>
  {#if children}
    {@render children()}
  {/if}
</div>
