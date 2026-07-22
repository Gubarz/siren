import {
  MonitorStart,
  MonitorStop,
  GetMonitorProviders,
  AddMonitorProvider,
  RemoveMonitorProvider,
} from '../../../wailsjs/go/gui/App.js';
import { responseField } from './normalize.js';

export { MonitorStart, MonitorStop };

export async function listProviders() {
  return responseField(await GetMonitorProviders(), 'Providers', []);
}

export async function addProvider(id, providerType, apiKey, apiPassword) {
  return AddMonitorProvider(id, providerType, apiKey, apiPassword);
}

export async function removeProvider(id) {
  return RemoveMonitorProvider(id);
}
