<script>
  import { onDestroy } from 'svelte'
  import SearchInput from '../ui/SearchInput.svelte'
  import EmptyState from '../ui/EmptyState.svelte'
  import ErrorState from '../ui/ErrorState.svelte'

  let {
    data = [],
    columns = [],
    keyField = '',
    selectable = 'none',
    selected = $bindable(new Set()),
    sortable = true,
    filterable = false,
    defaultSort = null,
    loading = false,
    error = null,
    emptyState = null,
    virtualize = false,
    onRowClick,
    onRowDblClick,
    onRowContextMenu,
    rowClass,
    children,
  } = $props()

  let filter = $state('')
  let sortKey = $state('')
  let sortDir = $state(1)
  let scrollEl = $state(null)
  let initializedSort = false
  let columnWidths = $state({})
  let resizingColumn = $state(null)

  const MIN_COLUMN_WIDTH = 28

  $effect(() => {
    const next = { ...columnWidths }
    let changed = false
    for (const col of columns) {
      if (!col?.key) continue
      if (next[col.key] == null && col.width) {
        next[col.key] = col.width
        changed = true
      }
    }
    if (changed) columnWidths = next
  })

  $effect.pre(() => {
    if (initializedSort) return
    sortKey = defaultSort?.key || ''
    sortDir = defaultSort?.dir === 'desc' ? -1 : 1
    initializedSort = true
  })

  function toggleSort(key) {
    if (sortKey === key) sortDir = -sortDir
    else { sortKey = key; sortDir = 1 }
  }

  function columnWidth(col) {
    return columnWidths[col.key] ?? col.width ?? null
  }

  function preferredColumnWidth(col) {
    return columnWidth(col) || 120
  }

  function totalColumnWidth() {
    return columns.reduce((sum, col) => sum + preferredColumnWidth(col), 0) || 1
  }

  function columnStyle(col) {
    const percent = (preferredColumnWidth(col) / totalColumnWidth()) * 100
    return `width: ${percent}%;`
  }

  function tableStyle() {
    return 'width: 100%; table-layout: fixed;'
  }

  function startColumnResize(event, col) {
    event.preventDefault()
    event.stopPropagation()
    resizingColumn = {
      key: col.key,
      startX: event.clientX,
      startWidth: columnWidth(col) || event.currentTarget.parentElement?.offsetWidth || 120,
      pointerID: event.pointerId,
    }
    window.addEventListener('pointermove', handleColumnResize)
    window.addEventListener('pointerup', stopColumnResize)
    window.addEventListener('pointercancel', stopColumnResize)
    try {
      event.currentTarget.setPointerCapture(event.pointerId)
    } catch {}
  }

  function handleColumnResize(event) {
    if (!resizingColumn) return
    const delta = event.clientX - resizingColumn.startX
    columnWidths = {
      ...columnWidths,
      [resizingColumn.key]: Math.max(MIN_COLUMN_WIDTH, resizingColumn.startWidth + delta),
    }
  }

  function stopColumnResize() {
    if (!resizingColumn) return
    resizingColumn = null
    window.removeEventListener('pointermove', handleColumnResize)
    window.removeEventListener('pointerup', stopColumnResize)
    window.removeEventListener('pointercancel', stopColumnResize)
  }

  const filtered = $derived(
    !filter ? data : data.filter((item) =>
      columns.some((col) =>
        String(item[col.key] || '').toLowerCase().includes(filter.toLowerCase())
      )
    )
  )

  const rows = $derived(
    [...filtered].sort((a, b) => {
      if (!sortKey) return 0
      let av = String(a[sortKey] || '').toLowerCase()
      let bv = String(b[sortKey] || '').toLowerCase()
      return av < bv ? -sortDir : av > bv ? sortDir : 0
    })
  )

  const ESTIMATED_ROW_HEIGHT = 32
  let rowVirtualizer = $state(null)

  $effect(() => {
    if (virtualize && scrollEl) {
      const init = async () => {
        const { createVirtualizer } = await import('@tanstack/svelte-virtual')
        rowVirtualizer = createVirtualizer({
          count: rows.length,
          getScrollElement: () => scrollEl,
          estimateSize: () => ESTIMATED_ROW_HEIGHT,
          overscan: 10,
        })
      }
      init()
    }
  })

  function renderCell(item, col) {
    if (col.cell) return col.cell(item)
    return item[col.key] ?? '-'
  }

  function handleRowClick(item, event) {
    if (selectable !== 'none') {
      const id = item[keyField]
      if (selectable === 'single') {
        selected = new Set([id])
      } else {
        const next = new Set(selected)
        if (event.ctrlKey || event.metaKey || event.shiftKey) {
          if (next.has(id)) next.delete(id)
          else next.add(id)
        } else {
          next.clear()
          next.add(id)
        }
        selected = next
      }
    }
    onRowClick?.(item, event)
  }

  function handleRowDblClick(item, event) {
    onRowDblClick?.(item, event)
  }

  function handleContextMenu(item, event) {
    event.preventDefault()
    onRowContextMenu?.(item, event)
  }

  function handleContainerKeydown(e) {
    const rows = scrollEl?.querySelectorAll('tbody tr[tabindex="0"]') || []
    const current = document.activeElement
    const idx = Array.from(rows).indexOf(current)
    if (e.key === 'ArrowDown') { e.preventDefault(); rows[Math.min(idx + 1, rows.length - 1)]?.focus() }
    else if (e.key === 'ArrowUp') { e.preventDefault(); rows[Math.max(idx - 1, 0)]?.focus() }
  }

  onDestroy(() => {
    stopColumnResize()
  })
</script>

<div class="flex flex-col h-full">
  {#if filterable}
    <div class="flex items-center gap-2 px-3 py-1 border-b border-line bg-surface-50 shrink-0">
      <SearchInput bind:value={filter} placeholder="Filter..." />
      <span class="text-xs text-fg-muted ml-auto">{rows.length} item{rows.length === 1 ? '' : 's'}</span>
    </div>
  {/if}

  <div class="flex-1 overflow-y-auto overflow-x-hidden min-h-0" bind:this={scrollEl}>
    {#if error}
      <ErrorState error={error} />
    {:else if rows.length === 0 && !loading}
      {#if emptyState}
        <EmptyState icon={emptyState.icon} title={emptyState.title} description={emptyState.description} />
      {:else}
        <EmptyState title="No items" />
      {/if}
    {:else}
      {#snippet rowSnippet(item, rowStyle = '')}
        <tr
          class="border-b border-table-line cursor-pointer {rowClass ? rowClass(item) : ''}"
          class:bg-row-selected={selected.has(item[keyField])}
          class:hover:bg-row-hover={!selected.has(item[keyField])}
          tabindex="0"
          style={rowStyle}
          onclick={(e) => handleRowClick(item, e)}
          ondblclick={(e) => handleRowDblClick(item, e)}
          oncontextmenu={(e) => handleContextMenu(item, e)}
        >
          {#each columns as col}
            <td
              class="px-2 py-1 truncate border-r border-table-line last:border-r-0"
              style={columnStyle(col)}
            >
              {#if children}
                {@render children(item, col)}
              {:else}
                {renderCell(item, col)}
              {/if}
            </td>
          {/each}
        </tr>
      {/snippet}

      <table class="border-collapse text-xs" style={tableStyle()} role="grid" onkeydown={handleContainerKeydown}>
        <colgroup>
          {#each columns as col}
            <col style={columnStyle(col)} />
          {/each}
        </colgroup>
        <thead>
          <tr>
            {#each columns as col}
              <th
                class="sticky top-0 bg-table-header text-fg-muted font-normal text-left px-2 py-1 border-b border-r border-table-line last:border-r-0 whitespace-nowrap relative group {sortable && col.sortable !== false ? 'cursor-pointer select-none' : ''}"
                class:hover:text-brand={sortable && col.sortable !== false}
                onclick={() => sortable && col.sortable !== false && toggleSort(col.key)}
                style={columnStyle(col)}
                role="columnheader"
                aria-sort={sortKey === col.key ? (sortDir === 1 ? 'ascending' : 'descending') : 'none'}
              >
                <span class="flex items-center gap-1">
                  <span class="truncate">{col.label}</span>
                  {#if sortKey === col.key}
                    <span class="text-xs">{sortDir === 1 ? '\u25B2' : '\u25BC'}</span>
                  {/if}
                </span>
                <button
                  type="button"
                  class="absolute right-0 top-0 h-full w-2 cursor-col-resize border-0 bg-transparent p-0 opacity-0 group-hover:opacity-100 hover:bg-brand focus-visible:opacity-100 focus-visible:bg-brand"
                  style="touch-action: none"
                  aria-label="Resize {col.label} column"
                  onpointerdown={(event) => startColumnResize(event, col)}
                  onclick={(event) => event.stopPropagation()}
                ></button>
              </th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#if virtualize && rowVirtualizer}
            {#each rowVirtualizer.getVirtualItems() as vRow}
              {@const item = rows[vRow.index]}
              {@render rowSnippet(item, `height: ${vRow.size}px; transform: translateY(${vRow.start}px); position: absolute; width: 100%;`)}
            {/each}
            <tr aria-hidden="true">
              <td colspan={columns.length} style="height: {rowVirtualizer.getTotalSize()}px; padding: 0; border: 0;"></td>
            </tr>
          {:else}
            {#each rows as item (item[keyField])}
              {@render rowSnippet(item)}
            {/each}
          {/if}
        </tbody>
      </table>
    {/if}
  </div>
</div>
