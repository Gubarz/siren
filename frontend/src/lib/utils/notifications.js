// Notification-preference helpers. Kept out of the config store because
// they're pure math + the Toasts component wants a single "should I show
// this?" call per event.

function minutes(hhmm) {
  const [h, m] = String(hhmm || '00:00').split(':').map((n) => parseInt(n, 10) || 0)
  return h * 60 + m
}

// isInDNDWindow — true if `now` (Date) falls inside the operator's DND
// window. Handles wrap-around (22:00 → 08:00 means "22:00 tonight through
// 08:00 tomorrow").
function isInDNDWindow(now, start, end) {
  const nowMin = now.getHours() * 60 + now.getMinutes()
  const startMin = minutes(start)
  const endMin = minutes(end)
  if (startMin === endMin) return false
  if (startMin < endMin) return nowMin >= startMin && nowMin < endMin
  return nowMin >= startMin || nowMin < endMin
}

export function shouldShowNotification(prefs, eventType, now = new Date()) {
  if (!prefs || prefs.enabled === false) return false
  const types = prefs.types || {}
  // Absent key defaults to allowed — the settings UI writes explicit
  // false to mute a type.
  if (types[eventType] === false) return false
  const dnd = prefs.dnd
  if (dnd?.enabled && isInDNDWindow(now, dnd.start, dnd.end)) return false
  return true
}
