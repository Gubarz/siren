<script>
  import { useSvelteFlow, useStore } from '@xyflow/svelte'

  const { fitView } = useSvelteFlow()
  const store = useStore()

  let { fitKey = '' } = $props()
  let lastFitKey = ''

  function graphKey() {
    return store.nodes.map((node) => node.id).sort().join('|')
  }

  $effect(() => {
    const ready = store.nodesInitialized
    const count = store.nodes?.length ?? 0
    const width = store.width
    const height = store.height
    if (!ready || count === 0 || width < 80 || height < 80) return

    const key = `${fitKey}:${graphKey()}@${Math.round(width)}x${Math.round(height)}`
    if (key === lastFitKey) return
    lastFitKey = key

    requestAnimationFrame(() => {
      requestAnimationFrame(() => fitView({ padding: 0.18 }))
    })
  })
</script>
