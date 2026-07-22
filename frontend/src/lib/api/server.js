import {
  GetCredentials,
  GetImplantBuilds,
  GetJobs,
  GetLoot,
  GetOperators,
  GetProfiles,
} from '../../../wailsjs/go/gui/App.js';
import { responseField } from './normalize.js';

export {
  AddCredential,
  DeleteImplantBuild,
  DeleteProfile,
  DownloadLoot,
  GenerateImplantAdvanced,
  GenerateImplantFromProfile,
  GetServerInfo,
  SaveProfileAdvanced,
  GetPivots,
  GetPivotListeners,
  GetScreenshotData,
  KillJob,
  RegenerateImplant,
  RemoveCredential,
  RemoveLoot,
  StartListener,
} from '../../../wailsjs/go/gui/App.js';

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
  return Object.entries(configs).map(([name, config]) => ({
    name,
    staged: Boolean(staged[name]),
    ...config,
  }));
}
