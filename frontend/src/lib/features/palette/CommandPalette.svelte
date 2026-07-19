<script>
  import Fuse from 'fuse.js'
  import Modal from '$components/patterns/Modal.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import Button from '$components/ui/Button.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import { getActions } from './actions.js'

  let { open = $bindable(false) } = $props()

  let query = $state('')
  let selectedIndex = $state(0)
  let inputEl = $state(null)
  let listEl = $state(null)
  let scrollSource = 'keyboard'

  $effect(() => {
    if (open) {
      query = ''
      selectedIndex = 0
      inputEl?.focus()
    }
  })

  let results = $derived.by(() => {
    const q = query.trim()
    if (!q) return []
    const actions = getActions()
    const fuse = new Fuse(actions, {
      keys: ['label', 'description', 'tags', 'section'],
      threshold: 0.4,
      distance: 200,
      ignoreLocation: true,
    })
    return fuse.search(q).map((r) => r.item)
  })

  let groupedResults = $derived.by(() => {
    const groups = new Map()
    for (const r of results) {
      const key = r.section || 'Other'
      if (!groups.has(key)) groups.set(key, [])
      groups.get(key).push(r)
    }
    const flat = []
    for (const [section, items] of groups) {
      flat.push({ header: section })
      for (const item of items) flat.push({ item })
    }
    return flat
  })

  function indexOfAction(action) {
    return results.indexOf(action)
  }

  $effect(() => {
    const lastIndex = Math.max(results.length - 1, 0)
    if (selectedIndex < 0) selectedIndex = 0
    if (selectedIndex > lastIndex) selectedIndex = lastIndex
  })

  $effect(() => {
    selectedIndex
    if (scrollSource !== 'keyboard') return
    const target = listEl?.querySelector(`[data-index="${selectedIndex}"]`)
    target?.scrollIntoView({ block: 'nearest' })
  })

  function execute(action) {
    if (!action) return
    action.on?.()
    open = false
  }

  function handleKeydown(e) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      if (results.length === 0) return
      scrollSource = 'keyboard'
      selectedIndex = Math.min(selectedIndex + 1, results.length - 1)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      if (results.length === 0) return
      scrollSource = 'keyboard'
      selectedIndex = Math.max(selectedIndex - 1, 0)
    } else if (e.key === 'Enter' && results[selectedIndex]) {
      e.preventDefault()
      execute(results[selectedIndex])
    } else if (e.key === 'Escape') {
      open = false
    }
  }
</script>

<Modal bind:open title="Command Palette" size="lg" position="top">
  <div>
    <div class="mb-3">
      <TextInput
        bind:element={inputEl}
        bind:value={query}
        placeholder="Type to search commands, agents, panels..."
        oninput={() => { selectedIndex = 0 }}
        onkeydown={handleKeydown}
      />
    </div>

    <div class="max-h-96 overflow-y-auto" bind:this={listEl}>
      {#if !query.trim()}
        <p class="text-fg-muted text-xs py-4 text-center">
          Type to search commands, views, panels, sessions...
        </p>
      {:else if results.length === 0}
        <p class="text-fg-muted text-sm py-4 text-center">No matches for "{query}"</p>
      {:else}
        {#each groupedResults as row}
          {#if row.header}
            <div class="px-2 pt-3 pb-1 text-2xs uppercase tracking-wider text-fg-muted">
              {row.header}
            </div>
          {:else}
            {@const i = indexOfAction(row.item)}
            {@const selected = i === selectedIndex}
            <Button
              color="alternative"
              data-index={i}
              class={`!w-full !flex !items-center !justify-start gap-3 !px-3 !py-2 text-sm text-left !rounded !border-0 !bg-transparent !text-fg hover:!bg-surface-200 ${selected ? '!bg-brand !text-fg-inverted hover:!bg-brand' : ''}`}
              onclick={() => execute(row.item)}
              onmouseenter={() => { scrollSource = 'mouse'; selectedIndex = i }}
            >
              {#if row.item.icon}
                <Icon name={row.item.icon} size={16} class="shrink-0 opacity-70" />
              {/if}
              <div class="flex-1 min-w-0">
                <div class="truncate">{row.item.label}</div>
                {#if row.item.description}
                  <div class="text-xs opacity-70 truncate">{row.item.description}</div>
                {/if}
              </div>
            </Button>
          {/if}
        {/each}
      {/if}
    </div>
  </div>
</Modal>
