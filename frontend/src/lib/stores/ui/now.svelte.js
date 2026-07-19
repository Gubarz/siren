class Now {
  value = $state(Math.floor(Date.now() / 1000))

  constructor() {
    setInterval(() => {
      this.value = Math.floor(Date.now() / 1000)
    }, 1000)
  }
}

export const now = new Now()
