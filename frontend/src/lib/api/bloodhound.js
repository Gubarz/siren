import {
  BloodHoundGetConfig,
  BloodHoundSaveConfig,
  BloodHoundTestConnection,
  BloodHoundStatus,
  BloodHoundCorrelate,
  BloodHoundAttackPaths,
  BloodHoundKerberoastTargets,
  BloodHoundIngestJobs,
  BloodHoundIngestJob,
  BloodHoundIngestLocalFile,
  BloodHoundStartCollection,
  BloodHoundCollections,
  BloodHoundMarkOwned,
  BloodHoundUnmarkOwned,
  BloodHoundSessions,
  BloodHoundLocalAdmins,
} from '../../../bindings/siren/cmd/gui/app.js';
import { responseField } from './normalize.js';

// Wire format matches the generated Config model exactly
// (serverUrl/tokenId/tokenKey/insecureTls).
function toWireConfig(config) {
  return {
    serverUrl: config.serverUrl ?? config.ServerUrl ?? '',
    tokenId: config.tokenId ?? config.TokenId ?? '',
    tokenKey: config.tokenKey ?? config.TokenKey ?? '',
    insecureTls: Boolean(config.insecureTls ?? config.InsecureTls),
  };
}

function normalizeStatus(status) {
  return {
    configured: Boolean(responseField(status, 'Configured', false)),
    connected: Boolean(responseField(status, 'Connected', false)),
    serverUrl: responseField(status, 'ServerUrl', ''),
    error: responseField(status, 'Error', ''),
  };
}

function normalizeConfigView(view) {
  return {
    serverUrl: responseField(view, 'ServerUrl', ''),
    tokenId: responseField(view, 'TokenId', ''),
    hasTokenKey: Boolean(responseField(view, 'HasTokenKey', false)),
    insecureTls: Boolean(responseField(view, 'InsecureTls', false)),
  };
}

export async function getBloodHoundConfig() {
  return normalizeConfigView(await BloodHoundGetConfig());
}

export async function saveBloodHoundConfig(config) {
  return BloodHoundSaveConfig(toWireConfig(config));
}

export async function testBloodHoundConnection(config) {
  return BloodHoundTestConnection(toWireConfig(config));
}

export async function getBloodHoundStatus() {
  return normalizeStatus(await BloodHoundStatus());
}

// Correlates agents to BloodHound entities. Returns {agentID: enrichment}.
export async function correlateAgents(agents) {
  const refs = (agents ?? []).map((agent) => ({
    id: agent.id ?? agent.ID ?? '',
    hostname: agent.hostname ?? agent.Hostname ?? '',
    username: agent.username ?? agent.Username ?? '',
    remoteAddress: agent.remoteAddress ?? agent.RemoteAddress ?? '',
  }));
  return (await BloodHoundCorrelate(refs)) ?? {};
}

// GraphDTO resolves directly ({nodes, edges}); no wrapper key to unwrap.
export async function getBloodHoundAttackPaths(objectId, maxPaths = 5) {
  return BloodHoundAttackPaths(objectId, maxPaths);
}

// Kerberoastable SPN accounts from the community query.
export async function getBloodHoundKerberoastTargets() {
  return (await BloodHoundKerberoastTargets()) ?? [];
}

export async function getBloodHoundIngestJobs() {
  return (await BloodHoundIngestJobs()) ?? [];
}

export async function getBloodHoundIngestJob(id) {
  return BloodHoundIngestJob(id);
}

// Uploads a local collection artifact (zip/json) to BloodHound.
export async function ingestBloodHoundLocalFile(path) {
  return BloodHoundIngestLocalFile(path);
}

// Starts a SharpHound/AzureHound collection on an agent; returns the run ID.
export async function startBloodHoundCollection(agentID, agentKind, agentOS, options) {
  return BloodHoundStartCollection(agentID, agentKind, agentOS, options);
}

export async function getBloodHoundCollections() {
  return (await BloodHoundCollections()) ?? [];
}

// Marks an entity owned in BloodHound via an object-ID selector on the
// built-in Owned tag; Unmark removes it. The backend invalidates the
// correlation cache so chips reflect the change on the next correlation.
export async function markBloodHoundOwned(objectId) {
  return BloodHoundMarkOwned(objectId);
}

export async function unmarkBloodHoundOwned(objectId) {
  return BloodHoundUnmarkOwned(objectId);
}

// Session relationships around an entity as GraphDTO; direction is decided
// server-side from entityKind ('Computer' vs principal kinds).
export async function getBloodHoundSessions(objectId, entityKind) {
  return BloodHoundSessions(objectId, entityKind);
}

// Local-admin relationships around an entity as GraphDTO, group-expanded
// server-side; direction is decided from entityKind.
export async function getBloodHoundLocalAdmins(objectId, entityKind) {
  return BloodHoundLocalAdmins(objectId, entityKind);
}
