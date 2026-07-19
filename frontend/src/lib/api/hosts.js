import {
  GetHost,
  GetHosts,
  RemoveHost,
  RemoveHostIOC,
} from '../../../wailsjs/go/main/App.js'
import { responseField } from './normalize.js'

export { RemoveHost, RemoveHostIOC }

export async function listHosts() {
  return responseField(await GetHosts(), 'Hosts', []).map(normalizeHost)
}

export async function getHost(hostUUID) {
  return normalizeHost(await GetHost(hostUUID))
}

function normalizeHost(host = {}) {
  const extensionData = responseField(host, 'ExtensionData', {})
  return {
    id: responseField(host, 'ID', ''),
    hostname: responseField(host, 'Hostname', ''),
    hostUUID: responseField(host, 'HostUUID', ''),
    osVersion: responseField(host, 'OSVersion', ''),
    locale: responseField(host, 'Locale', ''),
    firstContact: Number(responseField(host, 'FirstContact', 0)),
    iocs: responseField(host, 'IOCs', []).map(normalizeIOC),
    extensionData: Object.entries(extensionData || {}).map(([name, data]) => ({
      name,
      output: responseField(data, 'Output', ''),
    })),
    raw: host,
  }
}

function normalizeIOC(ioc = {}) {
  return {
    id: responseField(ioc, 'ID', ''),
    path: responseField(ioc, 'Path', ''),
    fileHash: responseField(ioc, 'FileHash', ''),
    raw: ioc,
  }
}
