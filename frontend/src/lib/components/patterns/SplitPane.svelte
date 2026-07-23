<script>
  let {
    size = $bindable(50),
    minSize = 10,
    maxSize = 90,
    orientation = 'horizontal',
    class: className = '',
    left,
    right,
  } = $props()

  let isResizing = $state(false)
  let containerEl = $state(null)
  let dividerEl = $state(null)

  function startResize(e) {
    e.preventDefault()
    isResizing = true

    dividerEl.setPointerCapture(e.pointerId)
  }

  function handleResize(e) {
    if (!isResizing || !containerEl) return
    const rect = containerEl.getBoundingClientRect()
    const total = orientation === 'horizontal' ? rect.width : rect.height
    const pos = orientation === 'horizontal' ? e.clientX - rect.left : e.clientY - rect.top
    let pct = (pos / total) * 100
    size = Math.max(minSize, Math.min(maxSize, pct))
  }

  function stopResize(e) {
    if (!isResizing) return
    isResizing = false
    dividerEl.releasePointerCapture(e.pointerId)
  }

  function handleKeydown(e) {
    if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown' && e.key !== 'ArrowLeft' && e.key !== 'ArrowRight') return
    e.preventDefault()
    const delta = e.key.includes('Up') || e.key.includes('Left') ? -3 : 3
    size = Math.max(minSize, Math.min(maxSize, size + delta))
  }

  const mainClass = $derived(orientation === 'horizontal' ? 'flex-row' : 'flex-col')
  const dividerClass = $derived(orientation === 'horizontal' ? 'w-1.5 h-full cursor-col-resize' : 'h-1.5 w-full cursor-row-resize')
</script>

<div
  bind:this={containerEl}
  class="flex {mainClass} w-full h-full overflow-hidden {className}"
  class:select-none={isResizing}
>
  <div 
    class="min-w-0 min-h-0 shrink-0 overflow-hidden flex flex-col" 
    class:pointer-events-none={isResizing}
    style={orientation === 'horizontal' ? `width: ${size}%` : `height: ${size}%`}
  >
    {@render left?.()}
  </div>
  
  <button
    bind:this={dividerEl}
    type="button"
    class="{dividerClass} bg-line hover:bg-brand shrink-0 transition-colors relative z-10 border-0 p-0"
    onpointerdown={startResize}
    onpointermove={handleResize}
    onpointerup={stopResize}
    onpointercancel={stopResize}
    onkeydown={handleKeydown}
    aria-label="Resize panes"
  ></button>
  
  <div class="min-w-0 min-h-0 overflow-hidden flex flex-col flex-1" class:pointer-events-none={isResizing}>
    {@render right?.()}
  </div>
</div>