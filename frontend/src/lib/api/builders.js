import {
  GetBuilders,
  GenerateExternalBuild,
  GetExternalBuildConfig,
  SaveExternalBuild,
} from '../../../bindings/siren/cmd/gui/app.js';
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
