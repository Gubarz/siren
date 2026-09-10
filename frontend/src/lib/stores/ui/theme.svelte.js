import { GetSystemTheme } from '../../api/discovery.js'

export const SYSTEM_THEME = 'system'
export const THEME_STORAGE_KEY = 'sliver-theme'
export const RESOLVED_THEME_STORAGE_KEY = 'sliver-resolved-theme'

const FALLBACK_THEME = 'dark'
const NATIVE_THEME_POLL_MS = 10000

// The webview media query follows the GTK theme (often light) while the
// backend detector reads the desktop color scheme, so a native answer wins
// over the browser query. The flag only gates the media-query fallback for
// plain-browser contexts where the binding never succeeds.
let nativeResolutionSucceeded = false

export function getBrowserSystemTheme() {
  try {
    if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) return 'dark'
    if (window.matchMedia?.('(prefers-color-scheme: light)').matches) return 'light'
    return FALLBACK_THEME
  } catch {
    return FALLBACK_THEME
  }
}

async function getNativeSystemTheme() {
  try {
    const t = await GetSystemTheme()
    return t === 'dark' || t === 'light' ? t : ''
  } catch {
    return ''
  }
}

async function resolveSystemTheme() {
  const nativeTheme = await getNativeSystemTheme()
  if (nativeTheme) {
    nativeResolutionSucceeded = true
    return nativeTheme
  }
  return getBrowserSystemTheme()
}

function getStoredResolvedTheme() {
  try {
    const t = localStorage.getItem(RESOLVED_THEME_STORAGE_KEY)
    return t === 'dark' || t === 'light' ? t : ''
  } catch {
    return ''
  }
}

function setResolvedTheme(resolvedTheme) {
  document.documentElement.setAttribute('data-theme', resolvedTheme)
  document.documentElement.setAttribute('data-system-theme', resolvedTheme)
  try {
    localStorage.setItem(RESOLVED_THEME_STORAGE_KEY, resolvedTheme)
  } catch {}
}

export function getStoredThemePreference() {
  try {
    return localStorage.getItem(THEME_STORAGE_KEY) || SYSTEM_THEME
  } catch {
    return SYSTEM_THEME
  }
}

export function applyThemePreference(preference = getStoredThemePreference()) {
  document.documentElement.setAttribute('data-theme-preference', preference)

  if (preference !== SYSTEM_THEME) {
    setResolvedTheme(preference)
    return preference
  }

  const initialTheme = getStoredResolvedTheme() || FALLBACK_THEME
  setResolvedTheme(initialTheme)

  void resolveSystemTheme().then((resolvedTheme) => {
    if (getStoredThemePreference() !== SYSTEM_THEME) return
    setResolvedTheme(resolvedTheme)
  })

  return initialTheme
}

export function watchSystemThemePreference() {
  let media = null
  try {
    media = window.matchMedia?.('(prefers-color-scheme: light)')
  } catch {}

  function handleMediaChange() {
    if (nativeResolutionSucceeded) return
    if (getStoredThemePreference() === SYSTEM_THEME) applyThemePreference(SYSTEM_THEME)
  }

  async function pollNativeTheme() {
    if (getStoredThemePreference() !== SYSTEM_THEME) return
    const nativeTheme = await getNativeSystemTheme()
    if (!nativeTheme) return
    nativeResolutionSucceeded = true
    if (getStoredThemePreference() !== SYSTEM_THEME) return
    if (document.documentElement.getAttribute('data-theme') !== nativeTheme) {
      setResolvedTheme(nativeTheme)
    }
  }

  const pollTimer = setInterval(pollNativeTheme, NATIVE_THEME_POLL_MS)
  if (media?.addEventListener) media.addEventListener('change', handleMediaChange)

  return () => {
    clearInterval(pollTimer)
    if (media?.removeEventListener) media.removeEventListener('change', handleMediaChange)
  }
}

class Theme {
  preference = $state(getStoredThemePreference())

  set(preference) {
    const next = preference || SYSTEM_THEME
    applyThemePreference(next)
    try {
      localStorage.setItem(THEME_STORAGE_KEY, next)
    } catch {}
    this.preference = next
  }
}

export const theme = new Theme()
