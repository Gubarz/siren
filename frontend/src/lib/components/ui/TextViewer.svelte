<script>
  let { data = '', filename = '' } = $props()

  let lines = $derived(data.split('\n'))
  let totalLines = $derived(lines.length)
  let lineHeight = 20

  let isBinary = $derived(/\.(bin|exe|dll|so|dylib|o|class|pyc|pdb|sys|drv|img|iso)$/i.test(filename))

  let containerEl = $state(null)
  let scrollTop = $state(0)
  let viewportHeight = $state(800)

  let visibleStart = $derived(Math.floor(scrollTop / lineHeight))
  let visibleCount = $derived(Math.ceil(viewportHeight / lineHeight) + 50)
  let visibleLines = $derived(lines.slice(visibleStart, Math.min(visibleStart + visibleCount, totalLines)))

  function onScroll() {
    if (!containerEl) return
    scrollTop = containerEl.scrollTop
    viewportHeight = containerEl.clientHeight
  }
</script>

{#if isBinary}
  <div class="px-3 py-2 m-2 bg-warning-500/10 border border-warning-500/30 rounded text-xs shrink-0 text-fg-muted">Binary file detected. Raw text display may be unreadable.</div>
{/if}

{#if totalLines > 50000}
  <div class="px-3 py-2 m-2 bg-warning-500/10 border border-warning-500/30 rounded text-xs shrink-0 text-fg-muted">Large file ({totalLines.toLocaleString()} lines). Only visible portion is rendered.</div>
{/if}

<div class="flex-1 min-h-0 overflow-y-auto font-mono text-xs leading-5" bind:this={containerEl} onscroll={onScroll}>
  <div style="height: {totalLines * lineHeight}px; position: relative;">
    {#each visibleLines as line, i}
      <div class="absolute left-0 right-0 flex px-2 whitespace-pre overflow-hidden text-ellipsis hover:bg-row-hover" style="top: {(visibleStart + i) * lineHeight}px; height: {lineHeight}px;">
        <span class="w-12 shrink-0 text-right select-none pr-3 text-fg-muted">{visibleStart + i + 1}</span>
        <span class="whitespace-pre overflow-hidden text-ellipsis flex-1">{line}</span>
      </div>
    {/each}
  </div>
</div>
