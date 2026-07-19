import {
  GetBuilders,
  GenerateExternalBuild,
  GetExternalBuildConfig,
  SaveExternalBuild,
} from '../../../wailsjs/go/main/App.js';
import { responseField } from './normalize.js';

export {
  GenerateExternalBuild,
  GetExternalBuildConfig,
  SaveExternalBuild,
};

export async function listBuilders() {
  const resp = await GetBuilders();
  return responseField(resp, 'Builders', []);
}
