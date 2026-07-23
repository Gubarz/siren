import { createResource } from '../lib/createResource.svelte.js'
import { GetAllComments } from '../../api/comments.js'

export const entityComments = createResource({
  name: 'entity-comments',
  fetch: async () => (await GetAllComments()) || {},
  events: ['comments-updated'],
})
