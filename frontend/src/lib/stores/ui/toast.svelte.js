let nextId = 0

class ToastStore {
  items = $state([])

  push({ variant = 'info', message = '', duration = 4000, action = null }) {
    const id = ++nextId
    this.items = [...this.items, { id, variant, message, action }]
    if (duration > 0) {
      setTimeout(() => this.dismiss(id), duration)
    }
    return id
  }

  dismiss(id) {
    this.items = this.items.filter((item) => item.id !== id)
  }

  clear() {
    this.items = []
  }
}

export const toast = new ToastStore()
