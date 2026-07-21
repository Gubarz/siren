<script>
  import { SvelteFlow, Background, Controls, MiniMap } from '@xyflow/svelte';
  import '@xyflow/svelte/dist/style.css';
  import GraphNode from './GraphNode.svelte';
  import GraphAutoFit from './GraphAutoFit.svelte';
  import Panel from '$components/patterns/Panel.svelte';
  import { config } from '$stores/config.svelte.js';
  import { agentColors } from '$stores/resources/agentColors.svelte.js';
  import { useResource } from '$stores/lib/createResource.svelte.js';
  import { pivotParentMap } from '../../utils/agents.js';
  import { layoutGraph, preservePositions, layoutSignature, topologySignature } from './layout.js';
  import {
    SERVER_W, SERVER_H,
    collectAgents, indexAgents, agentNode, addC2Links, addDiscoveryNodes,
  } from './nodeBuilders.js';

  let {
    sessions = [],
    beacons = [],
    pivotGraph = null,
    pivotListeners = [],
    discoveries = [],
    selectedAgentIDs = [],
    selectedDiscoveryKeys = [],
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
    lastLayoutSig = '';
    build();
  });

  const nodeTypes = { box: GraphNode };

  useResource(agentColors)
  let colorsByAgent = $derived(
    agentColors?.data && typeof agentColors.data === 'object' ? agentColors.data : {},
  )

  let nodes = $state.raw([]);
  let edges = $state.raw([]);
  let lastSig = '';
  let lastLayoutSig = '';
  let fitKey = $state('');
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

  function build() {
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
        parentBySession, allAgents, now, direction, selectedAgentIDs, colorsByAgent,
      }));
    }
    addC2Links(rawNodes, rawEdges, {
      allAgents, index, parentBySession, direction, pivotListeners,
    });
    addDiscoveryNodes(rawNodes, rawEdges, {
      allAgentIds: index.allAgentIds, discoveries, direction, selectedDiscoveryKeys,
    });

    const nextLayoutSig = layoutSignature(rawNodes, rawEdges);
    const laidOut = layoutGraph(rawNodes, rawEdges, direction);
    const preserve = nextLayoutSig === lastLayoutSig;
    nodes = preserve ? preservePositions(nodes, laidOut, rawEdges) : laidOut;
    edges = rawEdges;
    if (!preserve) fitKey = nextLayoutSig;
    lastLayoutSig = nextLayoutSig;
  }

  $effect(() => {
    const sig = topologySignature({ sessions, beacons, pivotGraph, pivotListeners, discoveries, now, colors: colorsByAgent });
    if (sig === lastSig) return;
    lastSig = sig;
    build();
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
        onselectionchange={handleSelectionChange}
        multiSelectionKey={['Control', 'Meta']}
        proOptions={{ hideAttribution: true }}
      >
        <GraphAutoFit {fitKey} />
        <Background gap={18} />
        <Controls />
        <MiniMap pannable zoomable />
      </SvelteFlow>
    </div>
  </div>
</Panel>
