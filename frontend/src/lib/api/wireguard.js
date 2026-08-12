import {
  StartWGListener,
  GenerateWGClientConfig,
  GenerateUniqueWGIP,
  WGStartSocks,
  WGStopSocks,
  WGListSocksServers,
  WGStartPortForward,
  WGStopPortForward,
  WGListForwarders,
} from '../../../bindings/siren/cmd/gui/app.js';
import { responseField } from './normalize.js';

export {
  StartWGListener,
  GenerateWGClientConfig,
  GenerateUniqueWGIP,
  WGStartSocks,
  WGStopSocks,
  WGStartPortForward,
  WGStopPortForward,
};

export async function listWGSocks(sessionID) {
  const resp = await WGListSocksServers(sessionID);
  return responseField(resp, 'Servers', []);
}

export async function listWGForwarders(sessionID) {
  const resp = await WGListForwarders(sessionID);
  return responseField(resp, 'Forwarders', []);
}
