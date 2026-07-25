const registry = new Map()

export function getRegisteredActions() {
  return [...registry.values()]
}

export function registerCommandActions(actions) {
  for (const action of actions) registry.set(action.id, action)
}
