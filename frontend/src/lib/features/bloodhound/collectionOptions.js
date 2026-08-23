// Pure option building for the collection modal. Kept component-free so the
// modal and the automation action editor share one sanitization contract.

export const COLLECTION_METHODS = [
  'Default', 'All', 'Session', 'LoggedOn', 'Group', 'CertServices',
  'LocalAdmin', 'RDP', 'DCOM', 'PSRemote', 'Trusts',
];

export function buildCollectionOptions(form) {
  const methods = (form.methods ?? []).filter((m) => m && String(m).trim() !== '');
  const flags = String(form.flags ?? '')
    .split(/\s+/)
    .map((f) => f.trim())
    .filter((f) => f !== '' && f !== '--Loop');
  const timeout = Number(form.timeoutMinutes ?? 15);
  return {
    collector: String(form.collector || 'sharphound').toLowerCase(),
    methods: methods.length > 0 ? methods : ['Default'],
    flags,
    domain: String(form.domain ?? '').trim(),
    timeoutSeconds: (Math.min(3600, Math.max(1, Number.isFinite(timeout) ? timeout : 15))) * 60,
    ingest: form.ingest !== false,
    loot: form.loot !== false,
  };
}
