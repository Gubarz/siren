import { onBloodhoundEvent } from '$api/runtime.js';
import { correlateAgents, getBloodHoundIngestJobs, getBloodHoundCollections } from '$api/bloodhound.js';

export const bloodhoundStore = $state({
  status: null,
  domains: [],
  enrichment: {},
  connected: false,
  ingestJobs: [],
  collections: [],
  collectionRequest: null,
});

function upsertJob(jobs, job) {
  const index = jobs.findIndex((j) => j.id === job?.id);
  if (index === -1) {
    jobs.unshift(job);
  } else {
    jobs[index] = { ...jobs[index], ...job };
  }
}

function upsertCollection(collections, state) {
  const index = collections.findIndex((c) => c.id === state?.id);
  if (index === -1) {
    collections.unshift(state);
  } else {
    collections[index] = { ...collections[index], ...state };
  }
}

let subscribed = false;

// Idempotent: every consumer calls this; the underlying wails event
// subscription is process-lifetime, so only the first call wires it.
export function subscribeBloodhound() {
  if (subscribed) return;
  subscribed = true;
  onBloodhoundEvent((event) => {
    const type = event?.type;
    const payload = event?.payload ?? {};
    if (type?.startsWith('bloodhound.collection.')) {
      upsertCollection(bloodhoundStore.collections, payload ?? {});
      return;
    }
    switch (type) {
      case 'bloodhound.status':
        bloodhoundStore.status = payload;
        bloodhoundStore.connected = Boolean(payload?.connected);
        break;
      case 'bloodhound.enrichment':
        bloodhoundStore.enrichment = payload ?? {};
        break;
      case 'bloodhound.synced':
        bloodhoundStore.domains = payload?.domains ?? [];
        bloodhoundStore.enrichment = payload?.enrichments ?? {};
        break;
      case 'bloodhound.ingest.job.started':
        upsertJob(bloodhoundStore.ingestJobs, payload);
        break;
      case 'bloodhound.ingest.job.progress':
      case 'bloodhound.ingest.job.completed':
      case 'bloodhound.ingest.job.failed':
        upsertJob(bloodhoundStore.ingestJobs, payload);
        break;
    }
  });
}

export async function refreshIngestJobs() {
  try {
    bloodhoundStore.ingestJobs = await getBloodHoundIngestJobs();
  } catch {
    // non-fatal; the panel shows its own empty state
  }
}

export async function refreshCollections() {
  try {
    bloodhoundStore.collections = await getBloodHoundCollections();
  } catch {
    // non-fatal; task cards simply stay stale
  }
}

// Asks the agent's BloodHound tab to open the collection modal. Consumed by
// AgentBloodhoundTab; used by the agent context menu.
export function requestCollection(agentID) {
  bloodhoundStore.collectionRequest = { agentID, at: Date.now() };
}

let debounceTimer = null;

// Debounced re-correlation for a snapshot of the agent list. Called whenever
// the sessions table changes; the backend caches per agent for 60s so the
// cost of duplicate calls is one HTTP round trip at most.
export function requestCorrelation(agents) {
  if (!bloodhoundStore.connected) return;
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(() => {
    correlateAgents(agents)
      .then((result) => {
        bloodhoundStore.enrichment = result ?? {};
      })
      .catch(() => {
        // non-fatal: chips simply stay stale until the next sync event
      });
  }, 500);
}
