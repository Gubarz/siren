import { ListSessionTasks, GetTaskCallPayloads } from '../../../bindings/siren/cmd/gui/app.js';

export async function listSessionTasks(sessionID, limit = 100) {
  return (await ListSessionTasks(sessionID, limit)) || [];
}

export async function getTaskCallPayloads(chainRef) {
  return (await GetTaskCallPayloads(chainRef)) || [];
}
