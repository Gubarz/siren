export function discoveryKey(device) {
  return device.key || `${device.agentID}|${device.ip}`
}
