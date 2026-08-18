import {
  GetCredentials,
  GetImplantBuilds,
  GetJobs,
  GetLoot,
  GetOperators,
  GetProfiles,
} from '../../../bindings/siren/cmd/gui/app.js';
import { responseField } from './normalize.js';

export {
  AddCredential,
  DeleteImplantBuild,
  DeleteProfile,
  DownloadLoot,
  GenerateImplantAdvanced,
  GenerateImplantFromProfile,
  GetLootContent,
  GetServerInfo,
  SaveProfileAdvanced,
  GetPivots,
  GetPivotListeners,
  PivotStopListener,
  GetScreenshotData,
  KillJob,
  RegenerateImplant,
  RemoveCredential,
  RemoveLoot,
  StartListener,
} from '../../../bindings/siren/cmd/gui/app.js';

export async function listCredentials() {
  return responseField(await GetCredentials(), 'Credentials', []);
}

export async function listJobs() {
  return responseField(await GetJobs(), 'Active', []);
}

export async function listLoot() {
  return responseField(await GetLoot(), 'Loot', []);
}

export async function listOperators() {
  return responseField(await GetOperators(), 'Operators', []);
}

export async function listProfiles() {
  return responseField(await GetProfiles(), 'Profiles', []);
}

export async function listImplantBuilds() {
  const response = await GetImplantBuilds();
  const configs = responseField(response, 'Configs', {});
  const staged = responseField(response, 'Staged', {});
  const resourceIds = responseField(response, 'ResourceIDs', {});
  return Object.entries(configs).map(([name, config]) => ({
    name,
    staged: Boolean(staged[name]),
    nonce: resourceIds[name]?.Value ?? resourceIds[name]?.value ?? null,
    ...config,
  }));
}
