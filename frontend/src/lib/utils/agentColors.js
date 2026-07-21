// Row-tint colors for the closed agent-color palette in internal/tags.
// Raw rgba values (no CSS declaration) so both the table (rowStyle) and
// the graph (node background) can compose them. Values are theme-agnostic
// tints over transparent.

export const AGENT_COLOR_BG = {
  red: 'rgba(239, 68, 68, 0.14)',
  orange: 'rgba(249, 115, 22, 0.14)',
  yellow: 'rgba(234, 179, 8, 0.16)',
  green: 'rgba(34, 197, 94, 0.14)',
  blue: 'rgba(59, 130, 246, 0.16)',
  purple: 'rgba(168, 85, 247, 0.16)',
  pink: 'rgba(236, 72, 153, 0.16)',
  gray: 'rgba(156, 163, 175, 0.18)',
}

export function agentColorStyle(name) {
  const color = AGENT_COLOR_BG[name]
  return color ? `background-color: ${color};` : ''
}
