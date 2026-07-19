export function matchesAutomationTarget(target, rule) {
  if (rule.targetKind && rule.targetKind !== 'any' && rule.targetKind !== target._kind) {
    return false;
  }

  const filter = rule.filter || {};
  return matchesPattern(target.OS, filter.os) &&
    matchesPattern(target.Arch, filter.arch) &&
    matchesPattern(target.Hostname, filter.hostname) &&
    matchesPattern(target.Username, filter.username) &&
    matchesPattern(target.Name, filter.name);
}

function matchesPattern(value, patterns) {
  if (!patterns || patterns.trim() === '*' || !patterns.trim()) return true;
  const normalized = String(value || '').toLowerCase();
  return patterns.split(',').some((pattern) => {
    const expression = pattern.trim()
      .replace(/[.+^${}()|[\]\\]/g, '\\$&')
      .replaceAll('*', '.*')
      .replaceAll('?', '.');
    return expression && new RegExp(`^${expression}$`, 'i').test(normalized);
  });
}
