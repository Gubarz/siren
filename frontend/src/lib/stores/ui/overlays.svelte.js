class Overlays {
  name = $state(null)
  props = $state({})
  instance = $state(0)

  open(name, props = {}) {
    this.instance += 1
    this.name = name
    this.props = props
  }

  close() {
    this.instance += 1
    this.name = null
    this.props = {}
  }
}

export const overlays = new Overlays()
