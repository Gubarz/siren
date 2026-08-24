<script>
  import { SvelteFlow, Background, Controls } from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';

  import BhGraphNode from './BhGraphNode.svelte';
  import { layoutGraph } from '../graph/layout.js';

  let { graph = null, onNodeClick = null, onEdgeClick = null } = $props();

  const nodeTypes = { box: BhGraphNode };

  const NODE_W = 168;
  const NODE_H = 40;

  let nodes = $state.raw([]);
  let edges = $state.raw([]);
  let hasGraph = $derived((graph?.nodes?.length ?? 0) > 0);

  $effect(() => {
    const rawNodes = (graph?.nodes ?? []).map((n) => ({
      id: n.id,
      w: NODE_W,
      h: NODE_H,
      data: {
        label: n.label || n.id,
        kind: n.kind || '',
        tierZero: Boolean(n.tierZero),
        owned: Boolean(n.owned),
      },
    }));
    const rawEdges = (graph?.edges ?? []).map((e, i) => ({
      id: `e_${e.source}_${e.target}_${i}`,
      source: e.source,
      target: e.target,
      label: e.label ?? '',
    }));
    nodes = layoutGraph(rawNodes, rawEdges, 'LR');
    edges = rawEdges;
  });

  function handleNodeClick(evt) {
    const node = evt?.node || evt?.detail?.node;
    if (node) onNodeClick?.(node);
  }

  function handleEdgeClick(evt) {
    const edge = evt?.edge || evt?.detail?.edge;
    if (edge) onEdgeClick?.(edge);
  }
</script>

{#if !hasGraph}
  <p class="text-xs text-fg-muted m-0">No attack paths available.</p>
{:else}
  <div class="relative overflow-hidden" style="height: 420px;">
    <SvelteFlow
      colorMode="dark"
      bind:nodes
      bind:edges
      {nodeTypes}
      fitView
      fitViewOptions={{ padding: 0.18 }}
      minZoom={0.2}
      proOptions={{ hideAttribution: true }}
      onnodeclick={handleNodeClick}
      onedgeclick={handleEdgeClick}
    >
      <Background gap={18} />
      <Controls />
    </SvelteFlow>
  </div>
{/if}
