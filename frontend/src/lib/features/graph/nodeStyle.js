import { colorTint } from '../../utils/agentColors.js'

// buildNodeStyle renders the inline style shared by graph nodes: an optional
// selection/accent border plus the row tint washed over the node's opaque
// panel background (gradient layer on top, panel color preserved underneath).
export function buildNodeStyle(data, selected, accentBorder = '') {
  const tint = colorTint(data.color)
  const border = selected ? 'border-color: var(--color-brand);' : accentBorder
  return border + (tint ? ` background-image: linear-gradient(${tint}, ${tint});` : '')
}
