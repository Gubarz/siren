export function implantFormat(f) {
  return ({ 0: 'shared lib', 1: 'shellcode', 2: 'executable', 3: 'service', 4: 'third-party' })[f] ?? f
}

export function formatBytes(bytes, options = {}) {
  const zeroText = options.zeroText ?? '0 B'
  const num = Number(bytes)
  if (!Number.isFinite(num) || num <= 0) {
    return zeroText
  }
  const units = options.binaryUnits
    ? ['B', 'KiB', 'MiB', 'GiB', 'TiB']
    : ['B', 'KB', 'MB', 'GB', 'TB']
  const k = 1024
  const i = Math.min(Math.floor(Math.log(num) / Math.log(k)), units.length - 1)
  const decimals = options.decimals ?? (i === 0 ? 0 : 1)
  const formatted = (num / Math.pow(k, i)).toFixed(decimals)
  return `${parseFloat(formatted)} ${units[i]}`
}

export function formatRelativeTime(timestamp, nowSec = Math.floor(Date.now() / 1000)) {
  if (!timestamp) return '-'
  let ts = timestamp
  if (typeof ts === 'string') {
    const num = Number(ts)
    if (Number.isFinite(num) && num > 0) {
      ts = num
    } else {
      ts = Math.floor(new Date(ts).getTime() / 1000)
    }
  }
  if (!Number.isFinite(ts) || isNaN(ts)) return '-'
  if (ts > 9999999999) {
    ts = Math.floor(ts / 1000)
  }
  const s = Math.max(0, nowSec - ts)
  if (s < 2) return 'just now'
  if (s < 60) return `${s}s ago`
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`
  return `${Math.floor(s / 86400)}d ago`
}

export function formatDateTime(timestamp) {
  if (!timestamp) return '-'
  let val = timestamp
  if (typeof val === 'number' && val <= 9999999999) {
    val = val * 1000
  }
  const d = new Date(val)
  return isNaN(d.getTime()) ? '-' : d.toLocaleString()
}
