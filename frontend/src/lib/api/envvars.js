import {
  GetEnvInfo,
  SetDataDirOverride,
  ClearDataDirOverride,
  SetLogDirOverride,
  ClearLogDirOverride,
} from '../../../wailsjs/go/gui/App.js';

let cachedEnvInfo = null;

export async function getEnvInfo() {
  cachedEnvInfo = await GetEnvInfo();
  return cachedEnvInfo;
}

export function getCachedEnvInfo() {
  return cachedEnvInfo;
}

export async function setDataDirOverride(dir) {
  await SetDataDirOverride(dir);
  cachedEnvInfo = await GetEnvInfo();
  return cachedEnvInfo;
}

export async function clearDataDirOverride() {
  await ClearDataDirOverride();
  cachedEnvInfo = await GetEnvInfo();
  return cachedEnvInfo;
}

export async function setLogDirOverride(dir) {
  await SetLogDirOverride(dir);
  cachedEnvInfo = await GetEnvInfo();
  return cachedEnvInfo;
}

export async function clearLogDirOverride() {
  await ClearLogDirOverride();
  cachedEnvInfo = await GetEnvInfo();
  return cachedEnvInfo;
}
