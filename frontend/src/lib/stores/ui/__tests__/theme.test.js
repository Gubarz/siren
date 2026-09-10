// @vitest-environment jsdom
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'

const mocks = vi.hoisted(() => ({
  GetSystemTheme: vi.fn(),
}))

vi.mock('../../../api/discovery.js', () => ({
  GetSystemTheme: mocks.GetSystemTheme,
}))

const RESOLVED_THEME_STORAGE_KEY = 'sliver-resolved-theme'

function installMatchMedia(initial = {}) {
  const state = { dark: false, light: false, ...initial }
  const listeners = new Set()

  window.matchMedia = vi.fn((query) => {
    const isLight = query.includes('light')
    return {
      media: query,
      get matches() {
        return isLight ? state.light : state.dark
      },
      onchange: null,
      addEventListener: (type, listener) => {
        if (type === 'change') listeners.add(listener)
      },
      removeEventListener: (type, listener) => {
        if (type === 'change') listeners.delete(listener)
      },
      addListener: () => {},
      removeListener: () => {},
      dispatchEvent: () => false,
    }
  })

  return {
    state,
    fireChange() {
      for (const listener of [...listeners]) listener({ matches: state.light })
    },
  }
}

function themeAttribute() {
  return document.documentElement.getAttribute('data-theme')
}

function flush() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

function loadTheme() {
  return import('../theme.svelte.js')
}

async function startSystemThemeWatcher(media) {
  const { applyThemePreference, watchSystemThemePreference } = await loadTheme()
  applyThemePreference()
  await vi.advanceTimersByTimeAsync(0)
  expect(themeAttribute()).toBe('dark')

  const stop = watchSystemThemePreference()
  media.state.dark = false
  media.state.light = true
  media.fireChange()
  await vi.advanceTimersByTimeAsync(0)
  return stop
}

describe('theme system resolution', () => {
  beforeEach(() => {
    vi.resetModules()
    mocks.GetSystemTheme.mockReset()
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
    document.documentElement.removeAttribute('data-theme-preference')
    document.documentElement.removeAttribute('data-system-theme')
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('prefers native dark over a light webview media query', async () => {
    localStorage.setItem('sliver-theme', 'system')
    installMatchMedia({ dark: false, light: true })
    mocks.GetSystemTheme.mockResolvedValue('dark')

    const { applyThemePreference } = await loadTheme()
    applyThemePreference()
    await flush()

    expect(themeAttribute()).toBe('dark')
    expect(localStorage.getItem(RESOLVED_THEME_STORAGE_KEY)).toBe('dark')
  })

  it('falls back to the browser media query when the native binding fails', async () => {
    localStorage.setItem('sliver-theme', 'system')
    installMatchMedia({ dark: false, light: true })
    mocks.GetSystemTheme.mockRejectedValue(new Error('no native theme provider'))

    const { applyThemePreference } = await loadTheme()
    applyThemePreference()
    await flush()

    expect(themeAttribute()).toBe('light')
    expect(localStorage.getItem(RESOLVED_THEME_STORAGE_KEY)).toBe('light')
  })

  it('never consults the native binding for an explicit preference', async () => {
    localStorage.setItem('sliver-theme', 'dark')
    installMatchMedia({ dark: false, light: true })
    mocks.GetSystemTheme.mockResolvedValue('light')

    const { applyThemePreference } = await loadTheme()
    applyThemePreference('dark')
    await flush()

    expect(themeAttribute()).toBe('dark')
    expect(localStorage.getItem(RESOLVED_THEME_STORAGE_KEY)).toBe('dark')
    expect(mocks.GetSystemTheme).not.toHaveBeenCalled()
  })

  it('applies the cached resolved theme synchronously for first paint', async () => {
    localStorage.setItem('sliver-theme', 'system')
    localStorage.setItem(RESOLVED_THEME_STORAGE_KEY, 'light')
    installMatchMedia({ dark: true, light: false })
    let resolveNative
    mocks.GetSystemTheme.mockReturnValue(new Promise((resolve) => { resolveNative = resolve }))

    const { applyThemePreference } = await loadTheme()
    applyThemePreference()

    expect(themeAttribute()).toBe('light')

    resolveNative('dark')
    await flush()

    expect(themeAttribute()).toBe('dark')
  })
})

describe('theme system watcher', () => {
  beforeEach(() => {
    vi.resetModules()
    mocks.GetSystemTheme.mockReset()
    localStorage.clear()
    document.documentElement.removeAttribute('data-theme')
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('polls the native theme and ignores webview media changes', async () => {
    localStorage.setItem('sliver-theme', 'system')
    const media = installMatchMedia({ dark: true, light: false })
    mocks.GetSystemTheme.mockResolvedValue('dark')

    const stop = await startSystemThemeWatcher(media)
    expect(themeAttribute()).toBe('dark')

    mocks.GetSystemTheme.mockResolvedValue('light')
    await vi.advanceTimersByTimeAsync(10000)
    expect(themeAttribute()).toBe('light')

    stop()
  })

  it('uses the media listener only when native resolution has never succeeded', async () => {
    localStorage.setItem('sliver-theme', 'system')
    const media = installMatchMedia({ dark: true, light: false })
    mocks.GetSystemTheme.mockRejectedValue(new Error('no native theme provider'))

    const stop = await startSystemThemeWatcher(media)
    expect(themeAttribute()).toBe('light')

    stop()
  })
})
