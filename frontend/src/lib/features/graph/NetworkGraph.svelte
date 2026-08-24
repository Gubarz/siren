<script>
  import { untrack } from 'svelte';
  import { SvelteFlow, Background, Controls, ControlButton, MiniMap } from '@xyflow/svelte';
  import { RotateCcw } from '@lucide/svelte';
  import '@xyflow/svelte/dist/style.css';
  import GraphNode from './GraphNode.svelte';
  import GraphAutoFit from './GraphAutoFit.svelte';
  import Panel from '$components/patterns/Panel.svelte';
  import { config } from '$stores/config.svelte.js';
  import { agentColors } from '$stores/resources/agentColors.svelte.js';
  import { agentTags } from '$stores/resources/agentTags.svelte.js';
  import { entityColors } from '$stores/resources/entityColors.svelte.js';
  import { entityComments } from '$stores/resources/entityComments.svelte.js';
  import { useResource } from '$stores/lib/createResource.svelte.js';
  import { pivotParentMap } from '../../utils/agents.js';
  import { layoutGraph, preservePositions, layoutSignature, topologySignature } from './layout.js';
  import {
    SERVER_W, SERVER_H,
    collectAgents, indexAgents, agentNode, addC2Links, addDiscoveryNodes,
  } from './nodeBuilders.js';
  import { addBloodhoundOverlay } from './bhOverlay.js';

  let {
    sessions = [],
    beacons = [],
    pivotGraph = null,
    pivotListeners = [],
    discoveries = [],
    selectedAgentIDs = [],
    selectedDiscoveryKeys = [],
    bloodhound = null,
    onBloodhoundToggle = () => {},
    embedded = false,
    onClose = () => {},
    onSelect = () => {},
    onInteract = () => {},
    onContextMenu = () => {},
    onSelectionChange = () => {},
    onDeviceContextMenu = () => {},
    direction = 'TB',
  } = $props();

  $effect(() => {
    // Rebuild layout whenever the parent flips the direction.
    void direction;
    untrack(() => build({ reset: true }));
  });

  const nodeTypes = { box: GraphNode };

  useResource(agentColors, agentTags, entityColors, entityComments)
  let colorsByAgent = $derived(
    agentColors?.data && typeof agentColors.data === 'object' ? agentColors.data : {},
  )
  let tagsByAgent = $derived(
    agentTags?.data && typeof agentTags.data === 'object' ? agentTags.data : {},
  )
  let colorsByEntity = $derived(
    entityColors?.data && typeof entityColors.data === 'object' ? entityColors.data : {},
  )
  let commentsByEntity = $derived(
    entityComments?.data && typeof entityComments.data === 'object' ? entityComments.data : {},
  )

  let nodes = $state.raw([]);
  let edges = $state.raw([]);
  let lastSig = '';
  let lastLayoutSig = '';
  let layoutCustomized = false;
  let fitKey = $state(0);
  let now = $state(Math.floor(Date.now() / 1000));
  let appZoom = $derived.by(() => {
    const zoom = Number(config?.zoom);
    return Number.isFinite(zoom) && zoom > 0 ? zoom : 1;
  });

  // Every-5s tick so "online / offline" indicators refresh without every
  // parent having to re-render. Cleanup lives on the effect return so we
  // don't need a paired onDestroy.
  $effect(() => {
    const timer = setInterval(() => { now = Math.floor(Date.now() / 1000); }, 5000);
    return () => clearInterval(timer);
  });

  function build({ reset = false } = {}) {
    const rawNodes = [{
      id: 'ts', w: SERVER_W, h: SERVER_H,
      data: { variant: 'server', label: 'Sliver Teamserver', direction },
    }];
    const rawEdges = [];

    const allAgents = collectAgents({ sessions, beacons });
    const index = indexAgents(allAgents);
    const parentBySession = pivotParentMap(pivotGraph);

    for (const impl of allAgents) {
      rawNodes.push(agentNode(impl, {
        parentBySession, allAgents, now, direction, selectedAgentIDs,
        colorsByAgent, tagsByAgent, commentsByEntity,
      }));
    }
    addC2Links(rawNodes, rawEdges, {
      allAgents, index, parentBySession, direction, pivotListeners,
    });
    addDiscoveryNodes(rawNodes, rawEdges, {
      allAgentIds: index.allAgentIds, discoveries, direction, selectedDiscoveryKeys,
      colorsByEntity, commentsByEntity,
    });
    if (bloodhound) {
      addBloodhoundOverlay(rawNodes, rawEdges, {
        agents: allAgents,
        enrichment: bloodhound.enrichment ?? {},
        showEdges: Boolean(bloodhound.showEdges),
        direction,
      });
    }

    const nextLayoutSig = layoutSignature(rawNodes, rawEdges);
    const laidOut = layoutGraph(rawNodes, rawEdges, direction);
    const structureChanged = nextLayoutSig !== lastLayoutSig;
    const shouldReset = reset || nodes.length === 0 || (!layoutCustomized && structureChanged);
    nodes = shouldReset ? laidOut : preservePositions(nodes, laidOut, rawEdges);
    edges = rawEdges;
    if (shouldReset) fitKey += 1;
    if (reset) layoutCustomized = false;
    lastLayoutSig = nextLayoutSig;
  }

  $effect(() => {
    const sig = topologySignature({
      sessions, beacons, pivotGraph, pivotListeners, discoveries, now,
      colors: colorsByAgent, tags: tagsByAgent, entityColors: colorsByEntity,
      comments: commentsByEntity,
      bloodhoundEnrichment: bloodhound?.enrichment ?? {},
      bloodhoundEdges: Boolean(bloodhound?.showEdges),
    });
    if (sig === lastSig) return;
    lastSig = sig;
    untrack(() => build());
  });

  function handleSelectionChange(selection) {
    onSelectionChange({
      agentIDs: selection.nodes
        .filter((node) => node.data?.variant === 'agent')
        .map((node) => node.id),
      deviceKeys: selection.nodes
        .filter((node) => node.data?.variant === 'device')
        .map((node) => node.data.key),
    });
  }

  function handleNodeDragStop(evt) {
    const movedNodes = evt?.nodes || evt?.detail?.nodes || [];
    if (movedNodes.length === 0) return;

    const movedPositions = new Map(
      movedNodes.map((node) => [node.id, { ...node.position }]),
    );
    nodes = nodes.map((node) => (
      movedPositions.has(node.id)
        ? { ...node, position: movedPositions.get(node.id) }
        : node
    ));
    layoutCustomized = true;
  }

  function resetLayout() {
    build({ reset: true });
  }

  function handleNodeClick(evt) {
    const node = evt?.node || evt?.detail?.node;
    const nativeEvent = evt?.event || evt?.detail?.event;
    if (node?.data?.variant === 'device') return;
    if (node && node.data && node.data.variant === 'agent') {
      if (!embedded || nativeEvent?.detail === 2) onInteract(node.id);
      else onSelect(node.id);
    }
  }

  function handleNodeContextMenu(evt) {
    let nativeEvent = evt?.event || evt?.detail?.event || evt;
    if (nativeEvent && typeof nativeEvent.preventDefault === 'function') {
      nativeEvent.preventDefault();
    }

    const node = evt?.node || evt?.detail?.node;
    if (node?.data?.variant === 'device') {
      onDeviceContextMenu(nativeEvent, node.data);
      return;
    }
    if (node && node.data && node.data.variant === 'agent') {
      const source = node.data.kind === 'beacon' ? beacons : sessions;
      const agent = source.find((item) => item.ID === node.id);
      if (agent) onContextMenu(nativeEvent, { ...agent, _kind: node.data.kind });
    }
  }
</script>

<Panel {embedded} onclose={onClose}>
  <div class="h-vh-75 w-full relative overflow-hidden" class:h-full={embedded}>
    <div
      class="min-w-full min-h-full"
      style="width: {appZoom * 100}%; height: {appZoom * 100}%; zoom: {1 / appZoom};"
    >
      <SvelteFlow
        colorMode="dark"
        bind:nodes
        bind:edges
        {nodeTypes}
        fitView
        fitViewOptions={{ padding: 0.18 }}
        minZoom={0.2}
        panOnDrag={[0, 1]}
        selectionOnDrag={false}
        onnodeclick={handleNodeClick}
        onnodecontextmenu={handleNodeContextMenu}
        onnodedragstop={handleNodeDragStop}
        onselectionchange={handleSelectionChange}
        multiSelectionKey={['Control', 'Meta']}
        proOptions={{ hideAttribution: true }}
      >
        <GraphAutoFit {fitKey} />
        <Background gap={18} />
        <Controls>
          {#if bloodhound}
            <ControlButton
              class={bloodhound.showEdges ? 'bg-brand! text-on-brand!' : ''}
              onclick={() => onBloodhoundToggle(!bloodhound.showEdges)}
              title={bloodhound.showEdges ? 'Hide BloodHound edges' : 'Show BloodHound edges'}
              aria-label="Toggle BloodHound edges"
              aria-pressed={bloodhound.showEdges}
            >
              <span style="font-size: 9px; font-weight: 700;">BH</span>
            </ControlButton>
          {/if}
          <ControlButton
            onclick={resetLayout}
            title="Reset layout"
            aria-label="Reset layout"
          >
            <RotateCcw style="fill: none;" />
          </ControlButton>
        </Controls>
        <MiniMap pannable zoomable />
      </SvelteFlow>
    </div>
  </div>
</Panel>
