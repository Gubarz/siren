const registry = new Map()

export function getRegisteredActions() {
  return [...registry.values()]
}
