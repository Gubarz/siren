<script>
  import { useSvelteFlow, useStore } from '@xyflow/svelte'

  const { fitView } = useSvelteFlow()
  const store = useStore()

  let { fitKey = 0 } = $props()
  let lastFitKey

  $effect(() => {
    const ready = store.nodesInitialized
    const count = store.nodes?.length ?? 0
    const width = store.width
    const height = store.height
    if (!ready || count === 0 || width < 80 || height < 80) return

    if (fitKey === lastFitKey) return
    lastFitKey = fitKey

    requestAnimationFrame(() => {
      requestAnimationFrame(() => fitView({ padding: 0.18 }))
    })
  })
</script>
