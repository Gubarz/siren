<script>
  let { data = '' } = $props()

  let bytes = $derived.by(() => {
    const arr = new Uint8Array(data.length)
    for (let i = 0; i < data.length; i++) arr[i] = data.charCodeAt(i)
    return arr
  })
  let totalRows = $derived(Math.ceil(bytes.length / 16))
  let rowHeight = 22

  let containerEl = $state(null)
  let scrollTop = $state(0)
  let viewportHeight = $state(600)

  let visibleStart = $derived(Math.floor(scrollTop / rowHeight))
  let visibleCount = $derived(Math.ceil(viewportHeight / rowHeight) + 5)
  let visibleRows = $derived.by(() => {
    const r = []
    const end = Math.min(visibleStart + visibleCount, totalRows)
    for (let i = visibleStart; i < end; i++) {
      const offset = i * 16
      const chunk = bytes.slice(offset, Math.min(offset + 16, bytes.length))
      const hex = Array.from(chunk).map((b) => b.toString(16).padStart(2, '0')).join(' ')
      const ascii = chunk.map((b) => (b >= 32 && b <= 126 ? String.fromCharCode(b) : '.')).join('')
      r.push({ offset: offset.toString(16).padStart(8, '0'), hex, ascii })
    }
    return r
  })

  function onScroll() {
    if (!containerEl) return
    const prevScroll = scrollTop
    scrollTop = containerEl.scrollTop
    if (scrollTop !== prevScroll) {
      viewportHeight = containerEl.clientHeight
    }
  }
</script>

<div class="flex flex-col flex-1 min-h-0 font-mono text-xs">
  <div class="flex gap-4 px-2 py-1 bg-chrome border-b border-line font-semibold shrink-0 text-fg-muted">
    <span class="w-20 shrink-0 text-fg-muted">Offset</span>
    <span class="flex-1 min-w-0 tracking-wide">00 01 02 03 04 05 06 07 08 09 0a 0b 0c 0d 0e 0f</span>
    <span class="w-40 shrink-0">ASCII</span>
  </div>
  <div class="flex-1 min-h-0 overflow-y-auto" bind:this={containerEl} onscroll={onScroll}>
    <div style="height: {totalRows * rowHeight}px; position: relative;">
      {#each visibleRows as row, i}
        <div class="absolute left-0 right-0 flex gap-4 px-2 items-center even:bg-row-hover" style="top: {(visibleStart + i) * rowHeight}px; height: {rowHeight}px;">
          <span class="w-20 shrink-0 text-fg-muted">{row.offset}</span>
          <span class="flex-1 min-w-0 tracking-wide">{row.hex}</span>
          <span class="w-40 shrink-0">{row.ascii}</span>
        </div>
      {/each}
    </div>
  </div>
</div>
