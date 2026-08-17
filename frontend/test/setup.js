// Browser API stubs for jsdom — flowbite-svelte and floating-ui touch these
// at import/mount time, and jsdom does not implement them.

const g = globalThis

if (g.window && !g.window.matchMedia) {
  g.window.matchMedia = (query) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })
}

if (typeof g.ResizeObserver === 'undefined') {
  g.ResizeObserver = class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
}

// jsdom does not implement the Popover API used by flowbite-svelte's Popper.
if (g.HTMLElement && !g.HTMLElement.prototype.showPopover) {
  g.HTMLElement.prototype.showPopover = function () {}
  g.HTMLElement.prototype.hidePopover = function () {}
}
