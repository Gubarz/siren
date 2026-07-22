import {
  Crackstations,
  CrackSubmitJob,
  CrackTaskByID,
  CrackTaskCancel,
  CrackFilesList,
  CrackFileDelete,
  CrackFileUploadFromPath,
} from '../../../wailsjs/go/gui/App.js';
import { responseField } from './normalize.js';

export {
  CrackSubmitJob,
  CrackTaskByID,
  CrackTaskCancel,
  CrackFileDelete,
  CrackFileUploadFromPath,
};

export async function listCrackstations() {
  const resp = await Crackstations();
  return responseField(resp, 'Crackstations', []);
}

export async function listCrackFiles() {
  const resp = await CrackFilesList();
  return responseField(resp, 'Files', []);
}
