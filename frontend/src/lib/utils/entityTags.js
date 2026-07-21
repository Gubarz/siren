import { agentColorStyle } from './agentColors.js'

export function entityKey(entityType, entityID) {
  const type = String(entityType || '').trim().toLowerCase()
  const id = String(entityID || '').trim()
  return type && id ? `${type}:${id}` : ''
}

export function entityColorStyle(colors, entityType, entityID) {
  const key = entityKey(entityType, entityID)
  return key ? agentColorStyle(colors?.[key]) : ''
}
