export function statusChip(task) {
  if (task?.evidence === 'verified') return { label: 'verified', class: 'text-success-500' };
  if (task?.evidence === 'failed') return { label: 'failed', class: 'text-danger-500' };
  const status = String(task?.status || '').trim();
  if (!status || status === 'attempted') return { label: 'attempted', class: 'text-fg-muted' };
  if (status === 'OK') return { label: 'ok', class: 'text-success-500' };
  return { label: status, class: 'text-danger-500' };
}

export function shortMethod(method) {
  const value = String(method || '');
  const at = value.lastIndexOf('/');
  return at >= 0 ? value.slice(at + 1) : (value || '-');
}

export function parsePayload(json) {
  if (!json) return null;
  try {
    return JSON.parse(json);
  } catch {
    return null;
  }
}

export function normalizeKeys(value) {
  if (Array.isArray(value)) return value.map(normalizeKeys);
  if (value && typeof value === 'object') {
    const out = {};
    for (const [key, item] of Object.entries(value)) out[key.toLowerCase()] = normalizeKeys(item);
    return out;
  }
  return value;
}

export function inferKind(method, data) {
  if (!data || typeof data !== 'object') return 'json';
  if (data.processes) return 'processes';
  if (data.files) return 'files';
  if (data.variables) return 'env';
  if (data.details) return 'services';
  if (data.entries) return 'netstat';
  if (data.data && String(method || '').endsWith('/Screenshot')) return 'image';
  if (data.stdout || data.stderr || data.response?.output) return 'text';
  return 'json';
}

function decodeBase64(value) {
  if (!value) return '';
  let binary;
  try {
    binary = atob(value);
  } catch {
    return String(value);
  }
  try {
    return new TextDecoder().decode(Uint8Array.from(binary, (ch) => ch.charCodeAt(0)));
  } catch {
    return binary;
  }
}

export function decodeTextPayload(data) {
  if (!data || typeof data !== 'object') return { text: '', err: '' };
  const chunks = [data.stdout, data.stderr].filter(Boolean).map(decodeBase64);
  const text = chunks.length ? chunks.join('') : decodeBase64(data.response?.output);
  const err = data.response?.err || data.err || '';
  if (!text && !err && data.path) return { text: String(data.path), err: '' };
  return { text, err: String(err) };
}

const KEY_LABELS = {
  sessionid: 'Session',
  ppid: 'PPID',
  pid: 'PID',
  fullinfo: 'Full info',
  displayname: 'Display name',
};

export function formatDurationNs(value) {
  if (typeof value !== 'number' && typeof value !== 'string') return String(value);
  const ns = Number(value);
  if (!Number.isFinite(ns)) return String(value);
  const totalSeconds = Math.round(ns / 1e9);
  if (totalSeconds < 60) return `${totalSeconds}s`;
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes < 60) return seconds === 0 ? `${minutes}m` : `${minutes}m ${seconds}s`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return remainingMinutes === 0 ? `${hours}h` : `${hours}h ${remainingMinutes}m`;
}

export function humanizeKey(key) {
  const value = String(key ?? '');
  if (KEY_LABELS[value]) return KEY_LABELS[value];
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : '';
}

function isEmptyValue(value) {
  if (value === '' || value === false || value === 0 || value === null || value === undefined) return true;
  if (Array.isArray(value)) return value.length === 0;
  if (value && typeof value === 'object') return Object.keys(value).length === 0;
  return false;
}

function formatSummaryValue(key, value) {
  if (key === 'timeout') return formatDurationNs(value);
  if (key === 'pid' || key === 'ppid') return String(value);
  if (Array.isArray(value)) {
    const joined = value
      .map((item) => (item && typeof item === 'object' ? JSON.stringify(item) : String(item)))
      .join(', ');
    return joined.length > 120 ? `${joined.slice(0, 120)}…` : joined;
  }
  if (value && typeof value === 'object') return JSON.stringify(value);
  return String(value);
}

export function requestSummary(data) {
  if (!data || typeof data !== 'object' || Array.isArray(data)) return [];
  const source = data.request && typeof data.request === 'object' && !Array.isArray(data.request) ? data.request : data;
  const entries = Object.entries(source);
  const ordered = [
    ...entries.filter(([key]) => key === 'sessionid'),
    ...entries.filter(([key]) => key !== 'sessionid'),
  ];
  const summary = [];
  for (const [key, value] of ordered) {
    if (isEmptyValue(value)) continue;
    summary.push({ label: humanizeKey(key), value: formatSummaryValue(key, value) });
  }
  return summary;
}
