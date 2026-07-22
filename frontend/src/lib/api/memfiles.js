import {
  MemfilesList,
  MemfilesAdd,
  MemfilesRemove,
} from '../../../wailsjs/go/gui/App.js';
import { responseField } from './normalize.js';

export { MemfilesAdd, MemfilesRemove };

export async function listMemfiles(sessionID) {
  const resp = await MemfilesList(sessionID);
  return {
    files: responseField(resp, 'Files', []),
    path: responseField(resp, 'Path', '/memfs'),
  };
}
