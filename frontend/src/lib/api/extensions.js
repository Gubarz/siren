import {
  RegisterExtensionFromPath,
  ListExtensions,
  CallExtension,
  RegisterWasmExtensionFromPath,
  ListWasmExtensions,
  ExecWasmExtension,
} from '../../../wailsjs/go/gui/App.js';
import { responseField } from './normalize.js';

export { RegisterExtensionFromPath, CallExtension, RegisterWasmExtensionFromPath, ExecWasmExtension };

export async function listExtensions(sessionID) {
  const resp = await ListExtensions(sessionID);
  return responseField(resp, 'Names', []);
}

export async function listWasmExtensions(sessionID) {
  const resp = await ListWasmExtensions(sessionID);
  return responseField(resp, 'Names', []);
}
