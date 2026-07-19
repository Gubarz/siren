class DialogStore {
  isOpen = $state(false)
  type = $state('') // 'alert', 'confirm', 'prompt'
  title = $state('')
  message = $state('')
  inputValue = $state('')
  resolve = null

  #open(type, title, message, inputValue = '') {
    return new Promise((resolve) => {
      this.type = type
      this.title = title
      this.message = message
      this.inputValue = inputValue
      this.resolve = resolve
      this.isOpen = true
    })
  }

  alert(message, title = 'Alert') {
    return this.#open('alert', title, message)
  }

  confirm(message, title = 'Confirm') {
    return this.#open('confirm', title, message)
  }

  prompt(message, title = 'Input Required', defaultValue = '') {
    return this.#open('prompt', title, message, defaultValue)
  }
}

export const dialog = new DialogStore()
