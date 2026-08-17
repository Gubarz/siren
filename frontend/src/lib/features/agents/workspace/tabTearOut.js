export function pointerScreenPoint(event, fallback = null) {
  const x = Number(event?.screenX)
  const y = Number(event?.screenY)
  if (Number.isFinite(x) && Number.isFinite(y) && (x !== 0 || y !== 0)) return { x, y }
  return fallback
}

export function isScreenPointOutsideWindow(point, hostWindow = window, margin = 0) {
  if (!point) return false
  const left = Number(hostWindow.screenX) || 0
  const top = Number(hostWindow.screenY) || 0
  const right = left + (Number(hostWindow.outerWidth) || Number(hostWindow.innerWidth) || 0)
  const bottom = top + (Number(hostWindow.outerHeight) || Number(hostWindow.innerHeight) || 0)
  return point.x <= left + margin || point.y <= top + margin ||
    point.x >= right - margin || point.y >= bottom - margin
}

export function isPointerAtWindowBoundary(event, point, hostWindow = window) {
  const clientX = Number(event?.clientX)
  const clientY = Number(event?.clientY)
  if (Number.isFinite(clientX) && Number.isFinite(clientY)) {
    return clientX <= 1 || clientY <= 1 ||
      clientX >= hostWindow.innerWidth - 2 || clientY >= hostWindow.innerHeight - 2
  }
  return isScreenPointOutsideWindow(point, hostWindow, 2)
}
