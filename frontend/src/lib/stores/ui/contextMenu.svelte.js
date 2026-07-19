class ContextMenu {
  isOpen = $state(false)
  x = $state(0)
  y = $state(0)
  target = $state(null)
  sections = $state([])

  open({ x, y, target = null, sections = [] }) {
    this.x = x
    this.y = y
    this.target = target
    this.sections = sections
    this.isOpen = true
  }

  close() {
    this.isOpen = false
  }
}

export const contextMenu = new ContextMenu()
