export const SYSTEM_THEME = 'system'
export const THEME_STORAGE_KEY = 'sliver-theme'

const FALLBACK_THEME = 'dark'

function hasNativeSystemThemeProvider() {
  return typeof window !== 'undefined' && typeof window.go?.main?.App?.GetSystemTheme === 'function'
}

export function getBrowserSystemTheme() {
  try {
    if (window.matchMedia?.('(prefers-color-scheme: dark)').matches) return 'dark'
    if (window.matchMedia?.('(prefers-color-scheme: light)').matches) return 'light'
    return FALLBACK_THEME
  } catch {
    return FALLBACK_THEME
  }
}

export function getSystemTheme() {
  return hasNativeSystemThemeProvider() ? FALLBACK_THEME : getBrowserSystemTheme()
}

async function getNativeSystemTheme() {
  try {
    const t = await window.go.main.App.GetSystemTheme()
    return t === 'dark' || t === 'light' ? t : ''
  } catch {
    return ''
  }
}

export function getStoredThemePreference() {
  try {
    return localStorage.getItem(THEME_STORAGE_KEY) || SYSTEM_THEME
  } catch {
    return SYSTEM_THEME
  }
}

export function resolveThemePreference(preference = getStoredThemePreference()) {
  return preference === SYSTEM_THEME ? getSystemTheme() : preference
}

export function applyThemePreference(preference = getStoredThemePreference()) {
  const resolvedTheme = resolveThemePreference(preference)
  document.documentElement.setAttribute('data-theme', resolvedTheme)
  document.documentElement.setAttribute('data-theme-preference', preference)
  document.documentElement.setAttribute('data-system-theme', resolvedTheme)
  if (preference === SYSTEM_THEME && hasNativeSystemThemeProvider()) {
    getNativeSystemTheme().then((nativeTheme) => {
      if (!nativeTheme || getStoredThemePreference() !== SYSTEM_THEME) return
      document.documentElement.setAttribute('data-theme', nativeTheme)
      document.documentElement.setAttribute('data-system-theme', nativeTheme)
    })
  }
  return resolvedTheme
}

export function watchSystemThemePreference() {
  let media
  try {
    media = window.matchMedia?.('(prefers-color-scheme: light)')
  } catch {
    return () => {}
  }

  if (!media?.addEventListener) return () => {}

  function handleChange() {
    if (getStoredThemePreference() === SYSTEM_THEME) applyThemePreference(SYSTEM_THEME)
  }

  media.addEventListener('change', handleChange)
  return () => media.removeEventListener('change', handleChange)
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
