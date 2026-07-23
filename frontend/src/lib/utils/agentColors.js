const COLORS = {
  red:    { hex: '#ef4444', alpha: 0.14 },
  orange: { hex: '#f97316', alpha: 0.14 },
  yellow: { hex: '#eab308', alpha: 0.16 },
  green:  { hex: '#22c55e', alpha: 0.14 },
  blue:   { hex: '#3b82f6', alpha: 0.16 },
  purple: { hex: '#a855f7', alpha: 0.16 },
  pink:   { hex: '#ec4899', alpha: 0.16 },
  gray:   { hex: '#9ca3af', alpha: 0.18 },
}

export const ROW_COLORS = Object.keys(COLORS)

export function colorHex(name) {
  return COLORS[name]?.hex
}

export function colorTint(name) {
  const c = COLORS[name]
  if (!c) return null
  const n = parseInt(c.hex.replace('#', ''), 16)
  return `rgba(${n >> 16 & 255}, ${n >> 8 & 255}, ${n & 255}, ${c.alpha})`
}

export function agentColorStyle(name) {
  const tint = colorTint(name)
  return tint ? `background-color: ${tint};` : ''
}
