export function parseTag(rawTag) {
  const str = String(rawTag || '').trim()
  const colonIdx = str.indexOf(':')
  if (colonIdx > 0 && colonIdx < str.length - 1) {
    const key = str.slice(0, colonIdx).trim()
    const value = str.slice(colonIdx + 1).trim()
    return { isTyped: true, key, value, raw: str }
  }
  return { isTyped: false, key: '', value: str, raw: str }
}

export function getTagCategoryStyle(key) {
  const k = (key || '').toLowerCase()
  if (k === 'env' || k === 'environment' || k === 'stage') {
    return {
      container: 'bg-blue-950/80 border-blue-500/40 text-blue-200',
      keyBg: 'bg-blue-600 text-white',
    }
  }
  if (k === 'role' || k === 'type' || k === 'service') {
    return {
      container: 'bg-teal-950/80 border-teal-500/40 text-teal-200',
      keyBg: 'bg-teal-600 text-white',
    }
  }
  if (k === 'prio' || k === 'priority' || k === 'crit' || k === 'severity' || k === 'risk') {
    return {
      container: 'bg-red-950/80 border-red-500/40 text-red-200',
      keyBg: 'bg-red-600 text-white',
    }
  }
  if (k === 'group' || k === 'team' || k === 'op' || k === 'campaign') {
    return {
      container: 'bg-purple-950/80 border-purple-500/40 text-purple-200',
      keyBg: 'bg-purple-600 text-white',
    }
  }
  if (k === 'status' || k === 'state') {
    return {
      container: 'bg-emerald-950/80 border-emerald-500/40 text-emerald-200',
      keyBg: 'bg-emerald-600 text-white',
    }
  }
  if (k === 'owner' || k === 'user' || k === 'operator') {
    return {
      container: 'bg-amber-950/80 border-amber-500/40 text-amber-200',
      keyBg: 'bg-amber-600 text-white',
    }
  }
  if (k === 'ip' || k === 'host' || k === 'domain' || k === 'net') {
    return {
      container: 'bg-indigo-950/80 border-indigo-500/40 text-indigo-200',
      keyBg: 'bg-indigo-600 text-white',
    }
  }
  return {
    container: 'bg-slate-800/80 border-slate-600/50 text-slate-200',
    keyBg: 'bg-slate-600 text-slate-100',
  }
}
