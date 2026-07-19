export function quote(v) {
  const s = String(v ?? '')
  if (s !== '' && !/[\s"'\\]/.test(s)) return s
  return `"${s.replaceAll('\\', '\\\\').replaceAll('"', '\\"')}"`
}

export function shellPath(tail) {
  if (!tail) return ''
  const m = tail.match(/^(?:--shell-path|-s)\s+(\S+)/)
  if (m) return m[1]
  if (tail.startsWith('-')) return ''
  return tail
}
