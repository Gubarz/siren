let openState = $state(false)
let entityTypeState = $state('')
let entityIDState = $state('')
let entityLabelState = $state('')
let entitiesState = $state([])

export const tagsModal = {
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
  get entities() {
    return entitiesState
  },
  openTags(type, id, label = '') {
    entityTypeState = type || 'entity'
    entityIDState = id || ''
    entityLabelState = label || id || ''
    entitiesState = []
    openState = true
  },
  openTagsForEntities(entities, label = '') {
    const list = Array.isArray(entities) ? entities.filter((e) => e?.type && e?.id) : []
    entitiesState = list
    entityTypeState = list[0]?.type || 'entity'
    entityIDState = list[0]?.id || ''
    entityLabelState = label || `${list.length} items`
    openState = true
  },
  close() {
    openState = false
  },
}
