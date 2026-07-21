let openState = $state(false)
let entityTypeState = $state('')
let entityIDState = $state('')
let entityLabelState = $state('')

export const commentsModal = {
  get open() {
    return openState
  },
  set open(val) {
    openState = val
  },
  get entityType() {
    return entityTypeState
  },
  get entityID() {
    return entityIDState
  },
  get entityLabel() {
    return entityLabelState
  },
  openComments(type, id, label = '') {
    entityTypeState = type || 'entity'
    entityIDState = id || ''
    entityLabelState = label || id || ''
    openState = true
  },
  close() {
    openState = false
  },
}
