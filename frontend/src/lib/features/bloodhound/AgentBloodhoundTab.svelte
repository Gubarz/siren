<script>
  import { onMount } from 'svelte';
  import Badge from '$components/ui/Badge.svelte';
  import Button from '$components/ui/Button.svelte';
  import BhGraph from '$features/bloodhound/BhGraph.svelte';
  import CollectionModal from '$features/bloodhound/CollectionModal.svelte';
  import CollectionTaskCard from '$features/bloodhound/CollectionTaskCard.svelte';
  import RelationSection from './RelationSection.svelte';
  import { uniqueEntities, sessionHeading, adminHeading } from './relations.js';
  import { bloodhoundStore, subscribeBloodhound, requestCorrelation, refreshCollections } from './bloodhound.svelte.js';
  import { actionsForEntity, actionsForEdge } from './actions.js';
  import { getBloodHoundAttackPaths, getBloodHoundKerberoastTargets, getBloodHoundSessions, getBloodHoundLocalAdmins, markBloodHoundOwned, unmarkBloodHoundOwned } from '$api/bloodhound.js';
  import { GetCommandCatalog } from '$api/console.js';
  import { commandModal } from '$stores/ui/commandModal.svelte.js';
  import { addToCase } from '$stores/ui/addToCase.svelte.js';
  import { tagsModal } from '$stores/ui/tagsModal.svelte.js';
  import { commentsModal } from '$stores/ui/commentsModal.svelte.js';
  import { sessions } from '$stores/resources/sessions.svelte.js';
  import { useResource } from '$stores/lib/createResource.svelte.js';
  import { errorMessage } from '$utils/errors.js';
  import { toast } from '$stores/ui/toast.svelte.js';

  useResource(sessions)

  let { sessionID = '' } = $props();

  onMount(() => {
    subscribeBloodhound();
    loadCatalog();
    loadKerberoastable();
    refreshCollections();
  });

  let showCollectionModal = $state(false);

  // Context-menu requests ("Collect BloodHound data…") land here after the
  // tab opens.
  $effect(() => {
    const req = bloodhoundStore.collectionRequest;
    if (req?.agentID === sessionID) {
      bloodhoundStore.collectionRequest = null;
      showCollectionModal = true;
    }
  });

  let agentCollections = $derived(
    (bloodhoundStore.collections ?? []).filter((c) => c.agentId === sessionID),
  );

  let catalog = $state([]);
  let kerberoastableIDs = $state(new Set());
  let clickedEdge = $state(null);

  async function loadCatalog() {
    try {
      const raw = await GetCommandCatalog('session');
      catalog = (raw?.groups ?? []).flatMap((g) => g.commands ?? []).filter((c) => c.name);
    } catch {
      catalog = [];
    }
  }

  async function loadKerberoastable() {
    try {
      const targets = await getBloodHoundKerberoastTargets();
      kerberoastableIDs = new Set((targets ?? []).map((t) => t.objectId).filter(Boolean));
    } catch {
      kerberoastableIDs = new Set();
    }
  }

  function runCommand(agentID, name) {
    const command = catalog.find((c) => c.name === name);
    if (!command) {
      toast.push({ variant: 'error', message: `Command not available: ${name}` });
      return;
    }
    commandModal.open({ command, targetIDs: [agentID], useSession: true });
  }

  let agent = $derived(sessions.data?.find((a) => a.ID === sessionID));
  let enrichment = $derived(bloodhoundStore.enrichment[sessionID] ?? null);
  let pathGraph = $state({ nodes: [], edges: [] });
  let loadingPaths = $state(false);
  let pathError = $state('');

  let entityActions = $derived(actionsForEntity({
    agent, enrichment, entity: enrichment?.entity,
    kerberoastableIDs, runCommand,
    addToCase, openTags: tagsModal.openTags.bind(tagsModal), openComments: commentsModal.openComments.bind(commentsModal),
  }));

  let togglingOwned = $state(false);

  // Marks/removes the resolved entity from BloodHound's built-in Owned tag.
  // The backend invalidates the correlation cache, so the optimistic flip
  // here is confirmed by the follow-up re-correlation.
  async function toggleOwned() {
    const entity = enrichment?.entity;
    if (!entity?.objectId || !agent) return;
    const wasOwned = Boolean(enrichment.owned);
    togglingOwned = true;
    try {
      if (wasOwned) {
        await unmarkBloodHoundOwned(entity.objectId);
      } else {
        await markBloodHoundOwned(entity.objectId);
      }
      bloodhoundStore.enrichment = {
        ...bloodhoundStore.enrichment,
        [sessionID]: { ...enrichment, owned: !wasOwned },
      };
      requestCorrelation([agent]);
      toast.push({ variant: 'success', message: wasOwned ? 'Removed from Owned' : 'Marked as Owned' });
    } catch (e) {
      toast.push({ variant: 'error', message: `Owned update failed: ${errorMessage(e)}` });
    } finally {
      togglingOwned = false;
    }
  }

  let edgeActions = $derived(actionsForEdge({
    agent, enrichment, entity: enrichment?.entity, edge: clickedEdge, runCommand,
  }));

  // Keep the backend correlation cache warm for this agent specifically.
  $effect(() => {
    if (agent && bloodhoundStore.connected) {
      requestCorrelation([agent]);
    }
  });

  // Load attack paths whenever the resolved entity changes; drop stale
  // responses if the entity moved on before the request landed.
  $effect(() => {
    const objectId = enrichment?.entity?.objectId;
    if (!objectId) {
      pathGraph = { nodes: [], edges: [] };
      pathError = '';
      return;
    }
    loadingPaths = true;
    pathError = '';
    getBloodHoundAttackPaths(objectId, 5)
      .then((g) => {
        if (bloodhoundStore.enrichment[sessionID]?.entity?.objectId !== objectId) return;
        pathGraph = { nodes: g?.nodes ?? [], edges: g?.edges ?? [] };
      })
      .catch((e) => {
        pathError = errorMessage(e);
        toast.push({ variant: 'error', message: `Attack paths: ${pathError}` });
      })
      .finally(() => {
        loadingPaths = false;
      });
  });

  let sessionsGraph = $state({ nodes: [], edges: [] });
  let loadingSessions = $state(false);
  let sessionsError = $state('');
  let showSessionsGraph = $state(false);
  let adminsGraph = $state({ nodes: [], edges: [] });
  let loadingAdmins = $state(false);
  let adminsError = $state('');
  let showAdminsGraph = $state(false);

  // Per-row actions reuse the entity bridge with the listed entity in place
  // of the resolved one.
  function rowActions(rowEntity) {
    return actionsForEntity({
      agent, enrichment, entity: rowEntity,
      kerberoastableIDs, runCommand,
      addToCase, openTags: tagsModal.openTags.bind(tagsModal), openComments: commentsModal.openComments.bind(commentsModal),
    });
  }

  // Load both relation graphs when the resolved entity changes; the
  // stale-response guard matches the attack-paths effect above.
  $effect(() => {
    const objectId = enrichment?.entity?.objectId;
    const kind = enrichment?.entity?.kind ?? '';
    if (!objectId) {
      sessionsGraph = { nodes: [], edges: [] };
      adminsGraph = { nodes: [], edges: [] };
      sessionsError = '';
      adminsError = '';
      return;
    }
    loadingSessions = true;
    sessionsError = '';
    getBloodHoundSessions(objectId, kind)
      .then((g) => {
        if (bloodhoundStore.enrichment[sessionID]?.entity?.objectId !== objectId) return;
        sessionsGraph = { nodes: g?.nodes ?? [], edges: g?.edges ?? [] };
      })
      .catch((e) => {
        sessionsError = errorMessage(e);
        toast.push({ variant: 'error', message: `Sessions: ${sessionsError}` });
      })
      .finally(() => {
        loadingSessions = false;
      });

    loadingAdmins = true;
    adminsError = '';
    getBloodHoundLocalAdmins(objectId, kind)
      .then((g) => {
        if (bloodhoundStore.enrichment[sessionID]?.entity?.objectId !== objectId) return;
        adminsGraph = { nodes: g?.nodes ?? [], edges: g?.edges ?? [] };
      })
      .catch((e) => {
        adminsError = errorMessage(e);
        toast.push({ variant: 'error', message: `Local admins: ${adminsError}` });
      })
      .finally(() => {
        loadingAdmins = false;
      });
  });
</script>

<div class="h-full overflow-auto p-6">
  <div class="max-w-7xl mx-auto flex flex-col gap-5">
    <div class="flex items-center gap-3 bg-panel border border-panel-border rounded-lg px-5 py-4">
      <div class="flex-1 min-w-0">
        <h3 class="m-0 mb-1 text-fg text-base">BloodHound — {agent?.Hostname || sessionID}</h3>
        <p class="text-xs mt-0 mb-0 text-fg-muted truncate">
          {agent?.Username ? `${agent.Username} · ` : ''}{agent?.OS || 'unknown OS'}
        </p>
      </div>
      <Button size="sm" color="primary" onclick={() => (showCollectionModal = true)}>
        Run collection
      </Button>
    </div>

    {#if !bloodhoundStore.connected}
      <div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
        <p class="text-xs text-fg-muted m-0">Connect BloodHound in Settings to enrich this agent.</p>
      </div>
    {:else if !enrichment?.entity?.objectId}
      <div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
        <p class="text-xs text-fg-muted m-0">No BloodHound entity matched for this agent.</p>
      </div>
    {:else}
      <div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
        <div class="flex items-center gap-2 mb-3">
          <h3 class="m-0 text-fg text-base">{enrichment.entity.name}</h3>
          {#if enrichment.entity.kind}
            <Badge>{enrichment.entity.kind}</Badge>
          {/if}
          {#if enrichment.tierZero}
            <Badge variant="danger">Tier 0</Badge>
          {/if}
          {#if enrichment.owned}
            <Badge variant="success">Owned</Badge>
          {/if}
          {#if enrichment.distanceToTierZero >= 0}
            <span class="text-xs text-fg-muted">{enrichment.distanceToTierZero} hop(s) to Tier-0</span>
          {/if}
          <span class="flex-1"></span>
          <Button size="sm" color={enrichment.owned ? 'alternative' : 'primary'} loading={togglingOwned} onclick={toggleOwned}>
            {enrichment.owned ? 'Remove Owned' : 'Mark Owned'}
          </Button>
        </div>
        {#if enrichment.entity.properties && Object.keys(enrichment.entity.properties).length > 0}
          <dl class="grid grid-cols-2 gap-x-4 gap-y-1 mb-3 m-0">
            {#each Object.entries(enrichment.entity.properties) as [key, value] (key)}
              <dt class="text-xs text-fg-muted col-span-1 truncate">{key}</dt>
              <dd class="text-xs m-0 col-span-1 truncate">{value}</dd>
            {/each}
          </dl>
        {/if}
        <div class="flex flex-wrap items-center gap-2 mb-3">
          {#each entityActions as action (action.label)}
            <Button
              size="sm"
              color={action.disabled ? 'alternative' : 'primary'}
              disabled={action.disabled}
              title={action.reason || ''}
              onclick={action.on}
            >
              {action.label}
            </Button>
          {/each}
        </div>
        {#if clickedEdge}
          <div class="flex flex-wrap items-center gap-2 mb-3">
            <span class="text-xs text-fg-muted">Edge {clickedEdge.label}:</span>
            {#each edgeActions as action (action.label)}
              <Button
                size="sm"
                color={action.disabled ? 'alternative' : 'primary'}
                disabled={action.disabled}
                title={action.reason || ''}
                onclick={action.on}
              >
                {action.label}
              </Button>
            {/each}
          </div>
        {/if}
        <div>
          <h4 class="m-0 mb-2 text-fg text-sm">Attack paths to Tier-0</h4>
          {#if loadingPaths}
            <p class="text-xs text-fg-muted m-0">Loading attack paths…</p>
          {:else if pathError}
            <p class="text-xs text-danger-500 m-0">{pathError}</p>
          {:else}
            <BhGraph graph={pathGraph} onEdgeClick={(edge) => { clickedEdge = edge; }} />
          {/if}
        </div>
        <div class="grid grid-cols-1 xl:grid-cols-2 gap-5">
          <RelationSection
            title={sessionHeading(enrichment.entity.kind)}
            entities={uniqueEntities(sessionsGraph)}
            graph={sessionsGraph}
            loading={loadingSessions}
            error={sessionsError}
            bind:showGraph={showSessionsGraph}
            actionsFor={rowActions}
            onEdgeClick={(edge) => { clickedEdge = edge; }}
          />
          <RelationSection
            title={adminHeading(enrichment.entity.kind)}
            entities={uniqueEntities(adminsGraph)}
            graph={adminsGraph}
            loading={loadingAdmins}
            error={adminsError}
            bind:showGraph={showAdminsGraph}
            actionsFor={rowActions}
            onEdgeClick={(edge) => { clickedEdge = edge; }}
          />
        </div>
      </div>
    {/if}

    <div class="bg-panel border border-panel-border rounded-lg px-5 py-5">
      <h4 class="m-0 mb-2 text-fg text-sm">Collections</h4>
      {#if agentCollections.length === 0}
        <p class="text-xs text-fg-muted m-0">No collections yet.</p>
      {:else}
        <div class="flex flex-col">
          {#each agentCollections as collection (collection.id)}
            <CollectionTaskCard {collection} />
          {/each}
        </div>
      {/if}
    </div>
  </div>
</div>

{#if showCollectionModal && agent}
  <CollectionModal {agent} onclose={() => (showCollectionModal = false)} />
{/if}
