<script>
  import { onMount } from 'svelte'
  import TagBadge from './TagBadge.svelte'

  let {
    tags = [],
    class: className = '',
  } = $props()

  let container = $state()
  let probe = $state()
  let visibleCount = $state(0)
  let frame = 0

  function availableWidth() {
    if (!container) return 0
    const style = getComputedStyle(container)
    return container.clientWidth -
      (Number.parseFloat(style.paddingLeft) || 0) -
      (Number.parseFloat(style.paddingRight) || 0)
  }

  function measure() {
    if (!container || !probe) return

    const width = availableWidth()
    const tagWidths = [...probe.querySelectorAll('[data-tag-probe]')]
      .map((element) => element.offsetWidth)
    const counterWidths = new Map(
      [...probe.querySelectorAll('[data-counter-probe]')]
        .map((element) => [Number(element.dataset.counterProbe), element.offsetWidth]),
    )
    const gap = Number.parseFloat(getComputedStyle(probe).columnGap) || 0

    let tagsWidth = tagWidths.reduce((sum, tagWidth) => sum + tagWidth, 0)
    if (tagsWidth + Math.max(0, tagWidths.length - 1) * gap <= width) {
      visibleCount = tagWidths.length
      return
    }

    for (let count = tagWidths.length - 1; count >= 0; count -= 1) {
      tagsWidth -= tagWidths[count]
      const hidden = tagWidths.length - count
      const used = tagsWidth +
        Math.max(0, count - 1) * gap +
        (count > 0 ? gap : 0) +
        (counterWidths.get(hidden) || 0)
      if (used <= width) {
        visibleCount = count
        return
      }
    }

    visibleCount = 0
  }

  function scheduleMeasure() {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(measure)
  }

  $effect(() => {
    void tags
    scheduleMeasure()
  })

  onMount(() => {
    const observer = new ResizeObserver(scheduleMeasure)
    if (container) observer.observe(container)
    scheduleMeasure()
    return () => {
      observer.disconnect()
      cancelAnimationFrame(frame)
    }
  })
</script>

{#if tags.length > 0}
  <div bind:this={container} class="relative min-w-0 {className}">
    <div class="flex flex-nowrap items-center gap-1 overflow-hidden">
      {#each tags.slice(0, visibleCount) as tag}
        <TagBadge {tag} />
      {/each}
      {#if visibleCount < tags.length}
        <span class="shrink-0 text-3xs text-fg-muted">+{tags.length - visibleCount}</span>
      {/if}
    </div>

    <div
      bind:this={probe}
      class="pointer-events-none absolute invisible flex w-max flex-nowrap items-center gap-1"
      aria-hidden="true"
    >
      {#each tags as tag}
        <span data-tag-probe><TagBadge {tag} /></span>
      {/each}
      {#each tags.keys() as index}
        <span
          class="text-3xs text-fg-muted"
          data-counter-probe={tags.length - index}
        >+{tags.length - index}</span>
      {/each}
    </div>
  </div>
{/if}
