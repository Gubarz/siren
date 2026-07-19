// Lightweight modal state holder. Every modal in the app follows the same
// pattern — an `open` flag and an optional payload — so instead of a pair
// of `$state` variables plus an `open<X>()` function in every component,
// use one `Modal` instance per modal:
//
//   const reconfigure = new Modal()
//   reconfigure.show(agent)      // in a context-menu handler
//   <ReconfigureModal bind:open={reconfigure.open} agent={reconfigure.data} />
//
// Fields are $state so `bind:open={modal.open}` participates in the rune
// reactivity graph. `show` is defined as an arrow-typed field, not a
// method, so callers can pass `modal.show` as a bare reference without
// binding `this`.
export class Modal {
  open = $state(false)
  data = $state(null)

  show = (payload = null) => {
    this.data = payload
    this.open = true
  }

  close = () => {
    this.open = false
  }
}
