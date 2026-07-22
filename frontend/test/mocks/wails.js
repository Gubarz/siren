import { vi } from 'vitest'

export function installWailsApp(methods = {}) {
  const app = new Proxy({ ...methods }, {
    get(target, prop) {
      if (!(prop in target)) target[prop] = vi.fn()
      return target[prop]
    },
  })
  globalThis.window = {
    ...(globalThis.window || {}),
    go: { gui: { App: app } },
  }
  return app
}

export function installWailsRuntime(methods = {}) {
  const runtime = new Proxy({ ...methods }, {
    get(target, prop) {
      if (!(prop in target)) target[prop] = vi.fn()
      return target[prop]
    },
  })
  globalThis.window = {
    ...(globalThis.window || {}),
    runtime,
  }
  return runtime
}
