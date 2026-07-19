// Global slot for the AddToCase modal so every panel can request "add
// this item to a case" without each mounting its own modal instance.
// Mounted once at App level via a system-tier root component (see
// AddToCaseRoot.svelte).

class AddToCase {
  isOpen = $state(false)
  collection = $state('')
  itemID = $state('')
  label = $state('')

  open({ collection = '', itemID = '', label = '' } = {}) {
    this.collection = collection
    this.itemID = itemID
    this.label = label
    this.isOpen = true
  }

  close() {
    this.isOpen = false
  }
}

export const addToCase = new AddToCase()
