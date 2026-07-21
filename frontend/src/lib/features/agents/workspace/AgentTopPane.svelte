<script>
  import { onMount } from 'svelte'
  import SessionsTable from '../SessionsTable.svelte'
  import NetworkGraph from '../../graph/NetworkGraph.svelte'
  import Icon from '$components/ui/Icon.svelte'
  import Button from '$components/ui/Button.svelte'
  import Select from '$components/ui/Select.svelte'
  import TextInput from '$components/ui/TextInput.svelte'
  import ReconfigureAgentModal from '../modals/ReconfigureAgentModal.svelte'
  import BeaconDetailModal from '../modals/BeaconDetailModal.svelte'
  import EditTagsModal from '../modals/EditTagsModal.svelte'
  import BulkActionBar from './BulkActionBar.svelte'

  import { sessions } from '$stores/resources/sessions.svelte.js'
  import { beacons } from '$stores/resources/beacons.svelte.js'
  import { pivots } from '$stores/resources/pivots.svelte.js'
  import { pivotListeners } from '$stores/resources/pivotListeners.svelte.js'
  import { discoveries } from '$stores/resources/discoveries.svelte.js'
  import { agentTags } from '$stores/resources/agentTags.svelte.js'
  import { agentColors } from '$stores/resources/agentColors.svelte.js'
  import { automation } from '$stores/resources/automation.svelte.js'
  import { useResource } from '$stores/lib/createResource.svelte.js'

  useResource(sessions, beacons, pivots, pivotListeners, discoveries, agentTags, agentColors, automation)
  import { selection } from '$stores/ui/selection.svelte.js'
  import { config } from '$stores/config.svelte.js'
  import { contextMenu } from '$stores/ui/contextMenu.svelte.js'
  import { commandModal } from '$stores/ui/commandModal.svelte.js'
  import { overlays } from '$stores/ui/overlays.svelte.js'
  import { addToCase } from '$stores/ui/addToCase.svelte.js'
  import { navigation } from '$stores/ui/navigation.svelte.js'
  import { workspaceState } from '$stores/workspaceState.svelte.js'
  import { agentTabs } from '$stores/agentTabs.svelte.js'
  import { dialog } from '$stores/ui/dialog.svelte.js'

  import { RemoveNetworkDiscoveries } from '../../../api/discovery.js'
  import { GetCommandCatalog } from '../../../api/console.js'
  import { SetAgentColor } from '../../../api/tags.js'
  import { errorMessage } from '../../../utils/errors.js'
  import { discoveryKey } from '../../../utils/discovery.js'

  import { createAgentActions } from './agentActions.js'
  import { createBulkActions } from './agentBulkActions.js'
  import { buildAgentContextSections, buildDiscoveryContextSections } from './agentContextActions.js'
  import { createAgentDataModel } from './useAgentData.svelte.js'
  import { useAgentShortcuts } from './useAgentShortcuts.svelte.js'
  import { Modal } from '$stores/ui/Modal.svelte.js'

  let agentFilter = $state(workspaceState.agentFilter || '')
  let filterInput = $state(null)

  $effect(() => {
    workspaceState.set('agentFilter', agentFilter)
  })

  // --- Auto-subscribed store reads via $store syntax + $derived ---
  let activeView = $derived(navigation.activeView)
  let viewMode = $derived(config?.agentViewMode || 'table')
  let graphDirection = $derived(config?.graphDirection || 'TB')

  let sessionData = $derived(sessions?.data || [])
  let beaconData = $derived(beacons?.data || [])
  let pivotData = $derived(pivots?.data || [])
  let pivotListenersData = $derived(pivotListeners?.data || [])
  let rawDiscoveredData = $derived(discoveries?.data || [])
  let selected = $derived(selection)
  let tagsByAgent = $derived(
    agentTags?.data && typeof agentTags.data === 'object' ? { ...agentTags.data } : {},
  )

  let sessionCategories = $state([])
  let beaconCategories = $state([])

  // --- Data engine: single call, every derived list back at once ---
  const dataModel = createAgentDataModel()
  let processed = $derived(
    dataModel.process(agentFilter, sessionData, beaconData, rawDiscoveredData, pivotData, tagsByAgent),
  )
  let discoveredData = $derived(processed.discoveredData)
  let combinedData = $derived(processed.combinedData)
  let filteredData = $derived(processed.filteredData)
  let filteredDiscoveries = $derived(processed.filteredDiscoveries)
  let graphSessions = $derived(processed.graphSessions)
  let graphBeacons = $derived(processed.graphBeacons)

  let selectedAgents = $derived(
    [...selected.agents]
      .map((id) => combinedData.find((a) => a.ID === id))
      .filter(Boolean),
  )


  // j/k/g/Enter/`/`/Escape shortcuts + cursor state all live in the hook.
  // Getters, not raw values, so the hook always reads the current
  // $derived snapshots when a shortcut fires.
  useAgentShortcuts({
    activeView: () => activeView,
    filteredData: () => filteredData,
    filterInput: () => filterInput,
  })

  // --- Effects ---
  // Only poll pivots/pivotListeners while the graph view is showing.
  $effect(() => {
    if (viewMode === 'graph') {
      pivots.startPolling(5000)
      pivotListeners.startPolling(5000)
      discoveries.refresh()
    } else {
      pivots.stopPolling()
      pivotListeners.stopPolling()
    }
  })

  // --- Lifecycle ---
  onMount(async () => {
    try {
      const [sessCatalog, beaconCatalog] = await Promise.all([
        GetCommandCatalog('session'),
        GetCommandCatalog('beacon').catch(() => null),
      ])
      sessionCategories = catalogToCategories(sessCatalog)
      beaconCategories = catalogToCategories(beaconCatalog || sessCatalog)
    } catch {
      sessionCategories = []
      beaconCategories = []
    }
  })

  function catalogToCategories(catalog) {
    return (catalog?.groups ?? [])
      .map((group) => ({
        category: group.title,
        commands: (group.commands ?? []).filter((c) => c.name),
      }))
      .filter((g) => g.commands.length > 0)
  }

  // --- Selection + action wiring ---
  function handleRowSelect({ id, additive }) {
    selection.select('agent', id, additive)
  }

  function selectedAgentIDsIncluding(agent) {
    return [
      ...[...selected.agents].filter((id) => id !== agent.ID),
      agent.ID,
    ]
  }

  function selectedAgentsIncluding(agent) {
    const ids = selectedAgentIDsIncluding(agent)
    return ids
      .map((id) => combinedData.find((a) => a.ID === id))
      .filter(Boolean)
  }

  function executeAgentCommand(command, targetIDs) {
    if (!command) return
    commandModal.open({ command, targetIDs, useSession: true })
  }

  const actions = createAgentActions({ dialog, discoveries, agentTabs, selectedAgentIDsIncluding })
  const bulk = createBulkActions({ dialog })
  const {
    runDiscovery, promptPingSweep, clearDiscoveries,
    killAgent, newShell, renameAgent, removeBeaconRecord,
    promoteBeacon, demoteSession, runAutomationRule,
  } = actions

  async function runBulk(fn) {
    await fn(selectedAgents)
    selection.clear()
  }

  // --- Modal state ---
  // Each Modal instance owns its own open flag + payload. The `show()`
  // method sets both in one shot, so context-menu handlers just point
  // at the method reference.
  const reconfigure = new Modal()
  const beaconDetail = new Modal()
  const editTags = new Modal()

  async function setAgentRowColor(agents, color) {
    try {
      await Promise.all(agents.map((a) => SetAgentColor(a.ID, color)))
      await agentColors.refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Row color failed: '), 'Row Color')
    }
  }

  // --- Context menus ---
  function openAgentContextMenu(nativeEvent, agent) {
    const isBeacon = agent._kind === 'beacon'
    const isWindows = (agent.OS || '').toLowerCase() === 'windows'
    contextMenu.open({
      x: nativeEvent.clientX,
      y: nativeEvent.clientY,
      target: agent,
      sections: buildAgentContextSections({
        agent, isBeacon, isWindows,
        catalog: isBeacon ? beaconCategories : sessionCategories,
        targetIDs: selectedAgentIDsIncluding(agent),
        targetAgents: selectedAgentsIncluding(agent),
        agentTabs,
        automationRules: automation.data || [],
        contextMenuHandlers: {
          openReconfigure: reconfigure.show,
          openEditTags: editTags.show,
          openBeaconDetail: (a) => beaconDetail.show(a.ID),
          promoteBeacon, demoteSession, newShell,
          runDiscovery, promptPingSweep, clearDiscoveries,
          renameAgent,
          runAutomationRule,
          setAgentRowColor,
          addToCase: (payload) => addToCase.open(payload),
          killAgent, removeBeaconRecord,
          executeAgentCommand,
        },
      }),
    })
  }

  function handleContextMenu({ event, session }) {
    openAgentContextMenu(event, session)
  }

  async function removeDiscoveries(device) {
    const keys = selected.devices.size > 0 ? [...selected.devices] : [discoveryKey(device)]
    const targets = keys
      .map((k) => discoveredData.find((d) => discoveryKey(d) === k))
      .filter(Boolean)
    const byAgent = new Map()
    for (const d of targets) {
      if (!d.ip) continue
      for (const observerID of (d.observerIDs || [d.agentID])) {
        if (!observerID) continue
        if (!byAgent.has(observerID)) byAgent.set(observerID, new Set())
        byAgent.get(observerID).add(d.ip)
      }
    }
    try {
      await Promise.all([...byAgent.entries()].map(([agentID, ips]) => RemoveNetworkDiscoveries(agentID, [...ips])))
      await discoveries.refresh()
    } catch (err) {
      await dialog.alert(errorMessage(err, 'Remove failed: '), 'Network Discovery')
    }
  }

  function openDiscoveryContextMenu(nativeEvent, device) {
    contextMenu.open({
      x: nativeEvent.clientX,
      y: nativeEvent.clientY,
      target: device,
      sections: buildDiscoveryContextSections({
        device,
        selectedCount: selected.devices.size,
        removeDiscoveries,
        clearDiscoveries,
      }),
    })
  }

  function handleDiscoveryContextMenu({ event, device }) {
    openDiscoveryContextMenu(event, device)
  }

  function handleDiscoverySelect({ key, additive }) {
    selection.select('device', key, additive)
  }

  function handleInteract(id) {
    agentTabs.openTab(id, 'console')
  }

  const viewOptions = [
    { value: 'table', label: 'Table' },
    { value: 'graph', label: 'Graph' },
  ]
</script>

<div class="flex shrink-0 items-center gap-2 px-3 py-1 border-b border-line bg-chrome text-fg-muted text-xs">
  <Button color="secondary" size="sm" icon="headphones" title="Listeners" onclick={() => overlays.open('listeners')} />
  <Button color="secondary" size="sm" icon="shield" title="Armory" onclick={() => overlays.open('armory')} />
  <Button color="secondary" size="sm" icon="images" title="Screenshot Gallery" onclick={() => overlays.open('gallery')} />
  <Button color="secondary" size="sm" icon="key" title="Credentials" onclick={() => overlays.open('credentials')} />
  <Button color="secondary" size="sm" icon="download" title="Loot" onclick={() => overlays.open('loot')} />
  <Button color="secondary" size="sm" icon="factory" title="Generate Implant" onclick={() => overlays.open('generate')} />

  <div class="ml-auto flex items-center gap-2">
  <BulkActionBar
  count={selected.agents.size}
  onkill={() => runBulk(bulk.bulkKill)}
  onrename={() => runBulk(bulk.bulkRenamePrefix)}
  onaddtag={() => runBulk(bulk.bulkAddTag)}
  onremovetag={() => runBulk(bulk.bulkRemoveTag)}
  onclear={() => selection.clear()}
/>
    <Icon name="search" />
    <div class="w-64">
      <TextInput size="sm" placeholder="Filter agents...  (/, j/k, Enter)" bind:value={agentFilter} bind:element={filterInput} />
    </div>
  </div>

  <label for="agent-view-mode">View</label>
  <div class="w-28">
    <Select
      id="agent-view-mode"
      size="sm"
      options={viewOptions}
      value={viewMode}
      onchange={(v) => config.set('agentViewMode', v)}
    />
  </div>
  {#if viewMode === 'graph'}
    <Button
      color="dark"
      size="xs"
      title="Toggle graph layout direction"
      onclick={() => config.set('graphDirection', graphDirection === 'TB' ? 'LR' : 'TB')}
    >
      {graphDirection === 'TB' ? '⇅ Vertical' : '⇆ Horizontal'}
    </Button>
  {/if}
  <span class="min-w-16 text-right">{filteredData.length} agent{filteredData.length === 1 ? '' : 's'}</span>
</div>



<div class="flex-1 min-h-0 overflow-hidden">
  {#if viewMode === 'table'}
    <SessionsTable
      data={filteredData}
      pivotGraph={pivotData}
      discoveries={discoveredData}
      selectedAgentIDs={[...selected.agents]}
      selectedDiscoveryKeys={[...selected.devices]}
      filterable={false}
      oncontextmenu={handleContextMenu}
      onselect={handleRowSelect}
      oninteract={handleInteract}
      ondiscoveryselect={handleDiscoverySelect}
      ondiscoverycontextmenu={handleDiscoveryContextMenu}
    />
  {:else}
    <NetworkGraph
      embedded
      direction={graphDirection}
      sessions={graphSessions}
      beacons={graphBeacons}
      pivotGraph={pivotData}
      pivotListeners={pivotListenersData}
      discoveries={filteredDiscoveries}
      selectedAgentIDs={[...selected.agents]}
      selectedDiscoveryKeys={[...selected.devices]}
      onSelect={() => {}}
      onInteract={(id) => handleInteract(id)}
      onContextMenu={(nativeEvent, agent) => openAgentContextMenu(nativeEvent, agent)}
      onSelectionChange={({ agentIDs, deviceKeys }) => selection.replace({ agents: agentIDs, devices: deviceKeys })}
      onDeviceContextMenu={(nativeEvent, device) => openDiscoveryContextMenu(nativeEvent, device)}
    />
  {/if}
</div>

<ReconfigureAgentModal bind:open={reconfigure.open} agent={reconfigure.data} />
<BeaconDetailModal bind:open={beaconDetail.open} beaconID={beaconDetail.data} />
<EditTagsModal bind:open={editTags.open} agent={editTags.data} />
