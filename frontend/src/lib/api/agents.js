import {
  GetBeaconTasks,
  GetBeacons,
  GetFileList,
  GetProcessList,
  GetSessions,
} from '../../../wailsjs/go/main/App.js';
import { responseField } from './normalize.js';

export {
  CancelBeaconTask,
  Chmod,
  CopyPath,
  DownloadFile,
  DownloadDirectory,
  DownloadMultipleTar,
  GetDownloadHistory,
  GetAllDownloadHistory,
  ClearDownloadHistory,
  GetAgentNotes,
  SaveAgentNote,
  GetBeaconTaskOutput,
  KillAgent,
  KillProcess,
  MakeDir,
  Pwd,
  RemovePath,
  RemoveBeacon,
  RenameAgent,
  RenamePath,
  TakeScreenshot,
  UploadFile,
  GrepFiles,
  GetServices,
  ListProxies,
  ListRportfwds,
  StartPortfwd,
  StartRportfwd,
  StartSocks,
  StopPortfwd,
  StopRportfwd,
  StopSocks,
  StartService,
  StopService,
  RemoveService,
  ViewRemoteFile,
  UploadFiles,
} from '../../../wailsjs/go/main/App.js';

export async function listBeacons() {
  return responseField(await GetBeacons(), 'Beacons', []);
}

export async function listSessions() {
  return responseField(await GetSessions(), 'Sessions', []);
}

export async function listBeaconTasks(beaconID) {
  return responseField(await GetBeaconTasks(beaconID), 'Tasks', []);
}

export async function listProcesses(sessionID, fullInfo = false) {
  return responseField(await GetProcessList(sessionID, fullInfo), 'Processes', []);
}

export async function listFiles(sessionID, path) {
  const response = await GetFileList(sessionID, path);
  return {
    files: responseField(response, 'Files', []),
    path: responseField(response, 'Path', path || '/'),
  };
}
export {
  CreateRegistryKey,
  DeleteRegistryEntry,
  ListRegistrySubKeys,
  ListRegistryValues,
  ReadRegistryValue,
  WriteRegistryValue,
} from '../../../wailsjs/go/main/App.js';
