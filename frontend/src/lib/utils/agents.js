function stripAddressPort(address) {
  const value = String(address || '').trim();
  const ipv6 = value.match(/^\[(.+)\]:\d+$/);
  if (ipv6) return ipv6[1];
  return value.replace(/:\d+$/, '');
}

export function pivotParentMap(pivotGraph) {
  const parents = new Map();

  function visit(entries, parentID = '') {
    for (const entry of entries || []) {
      const session = entry.Session || entry.session;
      const sessionID = session?.ID || session?.id;
      if (sessionID && parentID) parents.set(sessionID, parentID);
      visit(entry.Children || entry.children, sessionID || parentID);
    }
  }

  visit(pivotGraph?.Children || pivotGraph?.children);
  return parents;
}

export function agentRemoteAddress(agent, parents, agents = []) {
  const parentID = parents?.get(agent.ID);
  if (!parentID) return stripAddressPort(agent.RemoteAddress || agent.remoteAddress);

  const parent = agents instanceof Map
    ? agents.get(parentID)
    : agents.find((candidate) => candidate.ID === parentID);
  const parentAddress = stripAddressPort(parent?.RemoteAddress || parent?.remoteAddress);
  return parentAddress
    ? `${parentAddress} -> ${shortAgentID(parentID)}`
    : shortAgentID(parentID);
}

export function isAgentOnline(agent, nowSeconds = Math.floor(Date.now() / 1000)) {
  if (agent.IsDead || agent.isDead) return false;
  const kind = agent._kind || (agent.NextCheckin !== undefined ? 'beacon' : 'session');
  if (kind !== 'beacon') return true;

  const nextCheckin = Number(agent.NextCheckin ?? agent.nextCheckin ?? 0);
  if (nextCheckin <= 0) return false;

  const interval = Number(agent.Interval ?? agent.interval ?? 0) / 1e9;
  const jitter = Number(agent.Jitter ?? agent.jitter ?? 0) / 1e9;
  const grace = Math.max(15, interval + jitter);
  return nowSeconds <= nextCheckin + grace;
}

export function shortAgentID(id) {
  return String(id || '').split('-')[0];
}

// ID → agent lookup across sessions and beacons. Entries are annotated with
// `_kind` so consumers (e.g. ReconfigureAgentModal) get the same agent shape
// the table/graph paths hand out — raw store objects carry no `_kind`, and
// without it a beacon renders as a session. On collision the beacon entry
// wins, matching the resolution order the agent dropdowns have always used.
export function buildAgentMap(sessionsList, beaconsList) {
  const map = new Map();
  for (const session of sessionsList || []) map.set(session.ID, { ...session, _kind: 'session' });
  for (const beacon of beaconsList || []) map.set(beacon.ID, { ...beacon, _kind: 'beacon' });
  return map;
}

export function isHighPrivilege(username) {
  const lower = String(username || '').toLowerCase();
  const account = lower.split(/[\\/]/).pop() || '';
  if (!account) return false;
  return account === 'root'
    || account.endsWith('$')
    || lower.includes('system')
    || lower.includes('admin');
}

function hostIdentity(device) {
  const mac = String(device.mac || '').trim().toLowerCase().replaceAll('-', ':');
  return mac ? `mac:${mac}` : `ip:${device.ip}`;
}

export function dedupeDiscoveries(devices) {
  const hosts = new Set();
  const byMAC = new Map();
  const byIP = new Map();

  function mergeHosts(target, source) {
    if (!source || target === source) return target;
    for (const observation of source.observations) {
      if (!target.observations.some(
        (item) => item.agentID === observation.agentID && item.ip === observation.ip,
      )) {
        target.observations.push(observation);
      }
    }
    for (const observerID of source.observerIDs) {
      if (!target.observerIDs.includes(observerID)) target.observerIDs.push(observerID);
    }
    if (!target.hostname && source.hostname) target.hostname = source.hostname;
    if (!target.vendor && source.vendor) target.vendor = source.vendor;
    if (!target.osHint && source.osHint) {
      target.osHint = source.osHint;
      target.ttl = source.ttl;
    }
    if (source.lastSeen > target.lastSeen) {
      target.lastSeen = source.lastSeen;
      target.method = source.method || target.method;
    }
    for (const [key, host] of byMAC) {
      if (host === source) byMAC.set(key, target);
    }
    for (const [key, host] of byIP) {
      if (host === source) byIP.set(key, target);
    }
    hosts.delete(source);
    return target;
  }

  for (const device of devices || []) {
    const mac = String(device.mac || '').trim().toLowerCase().replaceAll('-', ':');
    const macMatch = mac ? byMAC.get(mac) : null;
    const ipMatch = byIP.get(device.ip);
    let existing = mergeHosts(macMatch || ipMatch, macMatch && ipMatch ? ipMatch : null);
    const observation = { agentID: device.agentID, ip: device.ip };
    if (!existing) {
      existing = {
        ...device,
        key: hostIdentity(device),
        observations: [observation],
        observerIDs: [device.agentID],
      };
      hosts.add(existing);
    } else {
      if (!existing.observations.some(
        (item) => item.agentID === observation.agentID && item.ip === observation.ip,
      )) {
        existing.observations.push(observation);
      }
      if (!existing.observerIDs.includes(device.agentID)) {
        existing.observerIDs.push(device.agentID);
      }
      if (!existing.hostname && device.hostname) existing.hostname = device.hostname;
      if (!existing.vendor && device.vendor) existing.vendor = device.vendor;
      if (!existing.osHint && device.osHint) {
        existing.osHint = device.osHint;
        existing.ttl = device.ttl;
      }
      if (device.lastSeen > existing.lastSeen) {
        existing.lastSeen = device.lastSeen;
        existing.method = device.method || existing.method;
      }
    }

    if (mac) {
      existing.mac = mac;
      existing.key = `mac:${mac}`;
      byMAC.set(mac, existing);
    }
    byIP.set(device.ip, existing);
  }

  return [...hosts].map((host) => ({
    ...host,
    observerIDs: [...host.observerIDs].sort(),
    observations: [...host.observations].sort((a, b) =>
      a.agentID.localeCompare(b.agentID) || a.ip.localeCompare(b.ip)),
  }));
}
